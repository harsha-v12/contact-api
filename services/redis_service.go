package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"contact-management/config"
)

// ImportJob represents the state of a CSV import job stored in Redis
type ImportJob struct {
	ImportID          string    `json:"import_id"`
	TotalRecords      int       `json:"total_records"`
	ProcessedRecords  int       `json:"processed_records"`
	SuccessfulRecords int       `json:"successful_records"`
	FailedRecords     int       `json:"failed_records"`
	Status            string    `json:"status"` // pending | processing | completed | failed
	CreatedAt         time.Time `json:"created_at"`
	CompletedAt       time.Time `json:"completed_at,omitempty"`
}

// CreateImportJob initializes the import job state in Redis
func CreateImportJob(ctx context.Context, importID string, totalRecords int) error {
	key := fmt.Sprintf("import:job:%s", importID)
	
	fields := map[string]interface{}{
		"import_id":          importID,
		"total_records":      totalRecords,
		"processed_records":  0,
		"successful_records": 0,
		"failed_records":     0,
		"status":             "pending",
		"created_at":         time.Now().Unix(),
	}

	err := config.RedisClient.HSet(ctx, key, fields).Err()
	if err != nil {
		return fmt.Errorf("failed to create redis import job: %w", err)
	}

	// Expire the job metadata in Redis after 7 days to clean up space automatically
	return config.RedisClient.Expire(ctx, key, 7*24*time.Hour).Err()
}

// UpdateImportProgress updates progress details of the import job in Redis
func UpdateImportProgress(ctx context.Context, importID string, processed, successful, failed int, status string) error {
	key := fmt.Sprintf("import:job:%s", importID)
	
	fields := map[string]interface{}{
		"processed_records":  processed,
		"successful_records": successful,
		"failed_records":     failed,
		"status":             status,
	}

	if status == "completed" || status == "failed" {
		fields["completed_at"] = time.Now().Unix()
	}

	err := config.RedisClient.HSet(ctx, key, fields).Err()
	if err != nil {
		return fmt.Errorf("failed to update redis import progress: %w", err)
	}

	// 1. Package the exact same progress into JSON
	progressJSON, _ := json.Marshal(map[string]interface{}{
		"import_id":          importID,
		"processed_records":  processed,
		"successful_records": successful,
		"failed_records":     failed,
		"status":             status,
	})

	// 2. Broadcast this JSON to a specific Redis channel for this import!
	config.RedisClient.Publish(ctx, "import_channel:"+importID, progressJSON)

	return nil
}

// GetImportJob retrieves the status and statistics of an import job
func GetImportJob(ctx context.Context, importID string) (*ImportJob, error) {
	key := fmt.Sprintf("import:job:%s", importID)
	
	val, err := config.RedisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis import job: %w", err)
	}
	
	// If the hash has no fields, the job doesn't exist
	if len(val) == 0 {
		return nil, fmt.Errorf("import job %s not found", importID)
	}

	var job ImportJob
	job.ImportID = val["import_id"]
	job.Status = val["status"]
	
	job.TotalRecords, _ = strconv.Atoi(val["total_records"])
	job.ProcessedRecords, _ = strconv.Atoi(val["processed_records"])
	job.SuccessfulRecords, _ = strconv.Atoi(val["successful_records"])
	job.FailedRecords, _ = strconv.Atoi(val["failed_records"])

	if createdAtUnix, err := strconv.ParseInt(val["created_at"], 10, 64); err == nil {
		job.CreatedAt = time.Unix(createdAtUnix, 0)
	}
	
	if val["completed_at"] != "" {
		if completedAtUnix, err := strconv.ParseInt(val["completed_at"], 10, 64); err == nil {
			job.CompletedAt = time.Unix(completedAtUnix, 0)
		}
	}

	return &job, nil
}
