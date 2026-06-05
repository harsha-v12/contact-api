package models

type CSVImportMessage struct {
	ImportID string `json:"import_id"`
	FilePath string `json:"file_path"`
}
