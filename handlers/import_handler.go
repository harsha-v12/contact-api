package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"contact-management/config"
	"contact-management/models"
	"contact-management/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/segmentio/kafka-go"
)

// @Summary Upload CSV File
// @Description Asynchronously imports a batch of contacts from a CSV file.
// @Tags import
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV file to upload"
// @Success 202 {object} map[string]interface{} "message, import_id, status"
// @Failure 400 {object} map[string]string "Validation error"
// @Security ApiKeyAuth
// @Router /contacts/import [post]
func UploadImportFileHandler(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "file is required"})
	}

	if filepath.Ext(file.Filename) != ".csv" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "only .csv files are supported"})
	}

	maxSizeMB := 50
	if envMax := os.Getenv("MAX_IMPORT_FILE_SIZE_MB"); envMax != "" {
		if m, err := strconv.Atoi(envMax); err == nil {
			maxSizeMB = m
		}
	}

	if file.Size > int64(maxSizeMB*1024*1024) {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": fmt.Sprintf("file size exceeds maximum limit of %d MB", maxSizeMB)})
	}

	importID := uuid.New().String()
	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, os.ModePerm)
	
	filePath := filepath.Join(uploadDir, fmt.Sprintf("%s.csv", importID))
	
	// Save the file
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to open uploaded file"})
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to create target file"})
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to save file"})
	}

	totalRecords, err := countCSVRows(filePath)
	if err != nil || totalRecords == 0 {
		os.Remove(filePath)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid or empty CSV file"})
	}

	ctx := c.Request().Context()
	err = services.CreateImportJob(ctx, importID, totalRecords)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to initialize import tracking"})
	}

	msg := models.CSVImportMessage{
		ImportID: importID,
		FilePath: filePath,
	}
	
	msgBytes, _ := json.Marshal(msg)
	
	err = config.KafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(importID),
		Value: msgBytes,
	})

	if err != nil {
		services.UpdateImportProgress(ctx, importID, 0, 0, 0, "failed")
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to queue import job"})
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"import_id": importID,
		"message":   fmt.Sprintf("Import job created. %d records will be processed in the background.", totalRecords),
		"status":    "pending",
	})
}

func countCSVRows(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, err
	}

	if len(records) <= 1 {
		return 0, nil
	}

	return len(records) - 1, nil
}

// @Summary Check Import Status
// @Description Poll the current status of an async import job.
// @Tags import
// @Produce json
// @Param import_id path string true "Import UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string "Job not found"
// @Security ApiKeyAuth
// @Router /contacts/import/{import_id} [get]
func GetImportStatusHandler(c echo.Context) error {
	importID := c.Param("import_id")
	
	job, err := services.GetImportJob(c.Request().Context(), importID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "import job not found"})
	}

	var percentage float64
	if job.TotalRecords > 0 {
		percentage = float64(job.ProcessedRecords) / float64(job.TotalRecords) * 100
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"import_id":             job.ImportID,
		"status":                job.Status,
		"total_records":         job.TotalRecords,
		"processed_records":     job.ProcessedRecords,
		"successful_records":    job.SuccessfulRecords,
		"failed_records":        job.FailedRecords,
		"completion_percentage": fmt.Sprintf("%.2f", percentage),
		"created_at":            job.CreatedAt,
		"completed_at":          job.CompletedAt,
	})
}




// flow of the structure import the status 
// flow structure of the import csv file

// User uploads CSV
//         ↓
// API saves CSV file
//         ↓
// Kafka Producer sends:
// {
//   import_id,
//   file_path
// }
//         ↓
// Kafka Topic
//         ↓
// Worker Consumer reads message
//         ↓
// processImportJob()
//         ↓
// Open CSV
//         ↓
// Read rows
//         ↓
// Check duplicates
//         ↓
// Insert contacts
//         ↓
// Create activities
//         ↓
// Update progress
//         ↓
// Mark completed
//         ↓
// Delete CSV file