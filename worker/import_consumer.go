package worker

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"contact-management/models"
	"contact-management/repository"
	"contact-management/services"

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

	f, err := os.Open(msg.FilePath)
	if err != nil {
		services.UpdateImportProgress(ctx, msg.ImportID, 0, 0, 0, "failed")
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		services.UpdateImportProgress(ctx, msg.ImportID, 0, 0, 0, "failed")
		return
	}

	if len(records) <= 1 {
		services.UpdateImportProgress(ctx, msg.ImportID, 0, 0, 0, "completed")
		return
	}

	processed := 0
	successful := 0
	failed := 0

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

		firstName := row[0]
		email := row[2]
		mobileNumber := row[3]

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
			FirstName:      row[0],
			LastName:       row[1],
			Email:          row[2],
			MobileNumber:   row[3],
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
			contact.Tags = []string{row[9]}
		}

		isDuplicate, err := repository.CheckDuplicate(ctx, contact.Email, contact.MobileNumber, nil)
		if isDuplicate {
			log.Printf("[Worker] Skipping row %d: Duplicate email/mobile", i)
			failed++
			continue
		}
		if err != nil {
			log.Printf("[Worker] CheckDuplicate error on row %d: %v", i, err)
		}

		err = repository.CreateContact(ctx, &contact)
		if err != nil {
			log.Printf("[Worker] CreateContact error on row %d: %v", i, err)
			failed++
		} else {
			successful++
			_ = repository.AddContactActivity(ctx, &models.ContactActivity{
				ID:           uuid.New(),
				ContactID:    contact.ID,
				ActivityType: "contact_created",
				Details:      "Contact record created via CSV Import",
				CreatedAt:    time.Now(),
			})
		}

		if processed%100 == 0 {
			services.UpdateImportProgress(ctx, msg.ImportID, processed, successful, failed, "processing")
		}
	}

	services.UpdateImportProgress(ctx, msg.ImportID, processed, successful, failed, "completed")
	
	// Clean up temporary file
	os.Remove(msg.FilePath)
}
