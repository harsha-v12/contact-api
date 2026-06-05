package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"contact-management/config"
	"contact-management/models"
	"contact-management/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func UploadImportFileHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	if filepath.Ext(file.Filename) != ".csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .csv files are supported"})
		return
	}

	maxSizeMB := 50
	if envMax := os.Getenv("MAX_IMPORT_FILE_SIZE_MB"); envMax != "" {
		if m, err := strconv.Atoi(envMax); err == nil {
			maxSizeMB = m
		}
	}

	if file.Size > int64(maxSizeMB*1024*1024) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file size exceeds maximum limit of %d MB", maxSizeMB)})
		return
	}

	importID := uuid.New().String()
	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, os.ModePerm)
	
	filePath := filepath.Join(uploadDir, fmt.Sprintf("%s.csv", importID))
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	totalRecords, err := countCSVRows(filePath)
	if err != nil || totalRecords == 0 {
		os.Remove(filePath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or empty CSV file"})
		return
	}

	ctx := c.Request.Context()
	err = services.CreateImportJob(ctx, importID, totalRecords)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize import tracking"})
		return
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue import job"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
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

func GetImportStatusHandler(c *gin.Context) {
	importID := c.Param("import_id")
	
	job, err := services.GetImportJob(c.Request.Context(), importID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "import job not found"})
		return
	}

	var percentage float64
	if job.TotalRecords > 0 {
		percentage = float64(job.ProcessedRecords) / float64(job.TotalRecords) * 100
	}

	c.JSON(http.StatusOK, gin.H{
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
