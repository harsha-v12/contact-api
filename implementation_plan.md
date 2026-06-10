# Production Tuning: Worker Pools & Batch Inserts

To make your API truly Enterprise-ready for millions of contacts, we need to upgrade the background worker from "Basic Mode" to "High-Performance Mode". This document explains the exact process and code changes required.

## The Two Major Upgrades

### 1. The Go "Worker Pool" (Protecting RAM)
Right now, if 10,000 files arrive, Go spawns 10,000 Goroutines (`go processImportJob`). This will crash the server's RAM. 
**The Solution:** We create a "Conveyor Belt" (a Go Channel). We spawn exactly 10 permanent workers. As thousands of files arrive, they are put on the belt, and the 10 workers safely process them one by one.

### 2. ClickHouse "Batch Inserts" (Protecting the Database)
Right now, if a CSV has 10,000 rows, the worker hits the database 10,000 separate times (`INSERT, INSERT, INSERT...`). ClickHouse hates this.
**The Solution:** We bundle the contacts into groups of 1,000. We hit the database exactly 10 times with massive `INSERT` statements.

---

## Proposed Changes

### `worker/import_consumer.go`
We will rewrite the main listener to use a Worker Pool pattern.

#### [MODIFY] import_consumer.go (The Listener)
```go
func StartImportConsumer() {
	// ... existing kafka setup ...

	// 1. Create the Conveyor Belt (Channel)
	jobs := make(chan models.CSVImportMessage, 100)

	// 2. Spawn exactly 10 permanent workers
	for w := 1; w <= 10; w++ {
		go func(workerID int) {
			for msg := range jobs {
				log.Printf("Worker %d starting job %s", workerID, msg.ImportID)
				processImportJob(msg)
			}
		}(w)
	}

	// 3. The Listener puts jobs on the belt
	for {
		m, err := r.ReadMessage(context.Background())
		// ... error handling ...
		var msg models.CSVImportMessage
		json.Unmarshal(m.Value, &msg)
		
		jobs <- msg // Put the job safely on the channel
	}
}
```

#### [MODIFY] import_consumer.go (The Batch Logic)
Inside `processImportJob()`, instead of inserting row by row, we will group them.
```go
	batchSize := 1000
	var batch []models.Contact

	for i, row := range records {
		// ... existing validation & duplicate checking ...

		contact := models.Contact{ /* ... mapped fields ... */ }
		batch = append(batch, contact) // Add to our bundle

		// When our bundle hits 1,000, send it to the database all at once!
		if len(batch) >= batchSize {
			repository.BatchCreateContacts(batch)
			successful += len(batch)
			batch = nil // Clear the bundle for the next 1,000
		}
	}

	// Insert any leftovers (e.g. the final 350 contacts)
	if len(batch) > 0 {
		repository.BatchCreateContacts(batch)
		successful += len(batch)
	}
```

---

### `repository/contact_list_repo.go`
We will add a new function specifically for High-Speed Batch Inserts.

#### [NEW FUNCTION] contact_list_repo.go
```go
func BatchCreateContacts(contacts []models.Contact) error {
	ctx := context.Background()
	
	// Start a massive bulk transaction
	batch, err := config.DB.PrepareBatch(ctx, "INSERT INTO contacts")
	if err != nil {
		return err
	}

	// Add all 1,000 contacts into the single transaction
	for _, c := range contacts {
		err := batch.Append(
			c.ID, c.FirstName, c.LastName, c.Email, c.MobileNumber,
			c.Gender, c.DateOfBirth, c.City, c.State, c.Country,
			c.Tags, c.Notes, c.CreatedAt, c.LastActivityAt, c.Version, c.IsDeleted,
		)
		if err != nil {
			return err
		}
	}

	// Execute exactly 1 database call!
	return batch.Send()
}
```

## User Review Required
If you are presenting this in your demo, these code snippets explain the exact logic of Enterprise Software Engineering. 

If you would like me to physically implement these advanced features into your Go code right now, simply click **Approve** and I will write the code for you!
