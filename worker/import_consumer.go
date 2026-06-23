package worker

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"contact-management/config"
	"contact-management/models"
	"contact-management/repository"
	"contact-management/services"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func StartImportConsumer() {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "contact-import"
	}
	groupID := os.Getenv("KAFKA_CONSUMER_GROUP")
	if groupID == "" {
		groupID = "contact-import-group"
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  groupID,
		MaxBytes: 10e6, // 10MB
		Dialer:   config.GetKafkaDialer(), // The FIX: Apply SASL/TLS authentication to the consumer!
	})

	fmt.Printf("[ImportConsumer] Listening on topic '%s' (group: %s)\n", topic, groupID)

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("[ImportConsumer] error while receiving message: %v\n", err)
			continue
		}

		var msg models.CSVImportMessage
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Printf("[ImportConsumer] failed to parse message: %v\n", err)
			continue
		}

		go processImportJob(msg)
	}
}

func processImportJob(msg models.CSVImportMessage) {
	ctx := context.Background()
	services.UpdateImportProgress(ctx, msg.ImportID, 0, 0, 0, "processing")

	// Parse S3 URI (e.g. s3://bucket/uploads/uuid.csv)
	var bucket, key string
	if strings.HasPrefix(msg.FilePath, "s3://") {
		parts := strings.SplitN(strings.TrimPrefix(msg.FilePath, "s3://"), "/", 2)
		if len(parts) == 2 {
			bucket = parts[0]
			key = parts[1]
		}
	} else {
		// Fallback for local testing if needed
		log.Printf("[Worker] Invalid S3 URI format: %s", msg.FilePath)
		services.UpdateImportProgress(ctx, msg.ImportID, 0, 0, 0, "failed")
		return
	}

	out, err := config.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("[Worker] Failed to download from S3: %v", err)
		services.UpdateImportProgress(ctx, msg.ImportID, 0, 0, 0, "failed")
		return
	}
	defer out.Body.Close()

	reader := csv.NewReader(out.Body)
	reader.FieldsPerRecord = -1 // Allow variable number of fields per row
	reader.LazyQuotes = true    // Be forgiving with quotes
	records, err := reader.ReadAll()
	if err != nil {
		services.UpdateImportProgress(ctx, msg.ImportID, 0, 0, 0, "failed")
		return
	}

	if len(records) <= 1 {
		services.UpdateImportProgress(ctx, msg.ImportID, 0, 0, 0, "completed")
		return
	}

	log.Printf("[Worker] Successfully opened CSV. Total lines: %d. Downloading Database Cache...", len(records))

	existingIdentifiers, err := repository.GetExistingIdentifiers(ctx)
	if err != nil {
		log.Printf("[Worker] Warning: Failed to load existing identifiers: %v", err)
		// Fallback to empty map if cache fails, or we could fail the import
	}
	log.Printf("[Worker] Cached %d unique identifiers from ClickHouse. Starting lightning fast loop...", len(existingIdentifiers))

	processed := 0
	successful := 0
	failed := 0

	var contactsBatch []*models.Contact
	var activitiesBatch []*models.ContactActivity

	flushBatch := func() {
		if len(contactsBatch) == 0 {
			return
		}	
		err := repository.BatchInsertContacts(ctx, contactsBatch)
		if err != nil {
			log.Printf("[Worker] BatchInsertContacts error: %v", err)
			failed += len(contactsBatch)
		} else {
			successful += len(contactsBatch)
			_ = repository.BatchAddContactActivities(ctx, activitiesBatch)
		}
		
		// same bucket used and clearing for the next batch without any creating the batch
		contactsBatch = contactsBatch[:0]
		activitiesBatch = activitiesBatch[:0]
	}

	for i, row := range records {
		if i == 0 {
			continue // Skip header
		}
		processed++
		if len(row) < 4 {
			failed++
			continue
		}
		// added for the checking the given data is correct format or not 

		// Sanitize data by trimming any accidental spaces from the CSV
		firstName := strings.TrimSpace(row[0])
		email := strings.TrimSpace(row[2])
		mobileNumber := strings.TrimSpace(row[3])

		if firstName == "" {
			log.Printf("[Worker] Skipping row %d: First Name is empty", i)
			failed++
			continue
		}

		if email == "" && mobileNumber == "" {
			log.Printf("[Worker] Skipping row %d: Both Email and Mobile are empty", i)
			failed++
			continue
		}

		if email != "" && !models.EmailRegex.MatchString(email) {
			log.Printf("[Worker] Skipping row %d: Invalid email format", i)
			failed++
			continue
		}

		if mobileNumber != "" && !models.MobileRegex.MatchString(mobileNumber) {
			log.Printf("[Worker] Skipping row %d: Invalid mobile number format", i)
			failed++
			continue
		}

		now := time.Now()
		contact := models.Contact{
			ID:             uuid.New(),
			FirstName:      firstName,
			LastName:       strings.TrimSpace(row[1]),
			Email:          email,
			MobileNumber:   mobileNumber,
			CreatedAt:      now,
			LastActivityAt: now,
			Version:        now,
			IsDeleted:      0,
		}

		if len(row) > 4 {
			contact.Gender = row[4]
		}
		if len(row) > 5 && row[5] != "" {
			if dob, err := time.Parse("2006-01-02", row[5]); err == nil {
				contact.DateOfBirth = dob
			}
		}
		if len(row) > 6 {
			contact.City = row[6]
		}
		if len(row) > 7 {
			contact.State = row[7]
		}
		if len(row) > 8 {
			contact.Country = row[8]
		}
		if len(row) > 9 && row[9] != "" {
			// Split by comma in case they provided multiple tags like "VIP,Customer"
			tags := strings.Split(row[9], ",")
			for i, t := range tags {
				tags[i] = strings.TrimSpace(t)
			}
			contact.Tags = tags
		}
		// Lightning Fast RAM Memory Lookup!
		isDuplicate := false
		if contact.Email != "" && existingIdentifiers[contact.Email] {
			isDuplicate = true
		}
		if contact.MobileNumber != "" && existingIdentifiers[contact.MobileNumber] {
			isDuplicate = true
		}

		if isDuplicate {
			// Do not log every duplicate to avoid spamming the terminal, but increment failed
			failed++
			continue
		}

		// Add new contacts to our RAM cache so we don't allow duplicates within the same CSV file!
		if contact.Email != "" {
			existingIdentifiers[contact.Email] = true
		}
		if contact.MobileNumber != "" {
			existingIdentifiers[contact.MobileNumber] = true
		}

		contactsBatch = append(contactsBatch, &contact)
		activitiesBatch = append(activitiesBatch, &models.ContactActivity{
			ID:           uuid.New(),
			ContactID:    contact.ID,
			ActivityType: "contact_created",
			Details:      "Contact record created via CSV Import",
			CreatedAt:    time.Now(),
		})

		if len(contactsBatch) >= 1000 {
			flushBatch()
		}

		// Update progress bar strictly every 100 rows!
		if processed%100 == 0 {
			log.Printf("[Worker] Milestone Reached: Processed %d records. Broadcasting to WebSocket...", processed)
			services.UpdateImportProgress(ctx, msg.ImportID, processed, successful, failed, "processing")
		}
	}

	// Flush any remaining contacts in the batch
	flushBatch()

	services.UpdateImportProgress(ctx, msg.ImportID, processed, successful, failed, "completed")
	log.Printf("[Worker] Finished Import %s. Total: %d, Success: %d, Failed: %d\n", msg.ImportID, processed, successful, failed)
}
