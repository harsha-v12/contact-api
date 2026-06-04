package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

var DB clickhouse.Conn

// ConnectDB initializes the connection to ClickHouse using native protocol with optional TLS
func ConnectDB() {
	host := getEnv("CLICKHOUSE_HOST", "localhost")
	port := getEnv("CLICKHOUSE_PORT", "9000")
	addr := fmt.Sprintf("%s:%s", host, port)

	username := getEnv("CLICKHOUSE_USER", "default")
	password := getEnv("CLICKHOUSE_PASSWORD", "")
	dbName := getEnv("CLICKHOUSE_DB", "default")
	secure := getEnv("CLICKHOUSE_SECURE", "false")

	var tlsConfig *tls.Config
	if secure == "true" {
		tlsConfig = &tls.Config{} // Enable secure TLS
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr:     []string{addr},
		Protocol: clickhouse.Native,
		TLS:      tlsConfig,
		Auth: clickhouse.Auth{
			Database: dbName,
			Username: username,
			Password: password,
		},
		DialTimeout:     30 * time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		panic(fmt.Errorf("failed to open clickhouse connection: %w", err))
	}

	if err := conn.Ping(context.Background()); err != nil {
		panic(fmt.Errorf("failed to ping clickhouse: %w", err))
	}

	DB = conn
	fmt.Printf("Connected to ClickHouse database successfully (%s, TLS: %s) ✅!\n", addr, secure)
}

// CreateTables executes DDL queries to verify and create tables if they do not exist
func CreateTables() {
	ctx := context.Background()

	// 1. Active and soft-deleted contacts table using ReplacingMergeTree
	createContactsTable := `
	CREATE TABLE IF NOT EXISTS contacts (
		id UUID,
		first_name String,
		last_name String,
		email String,
		mobile_number String,
		gender String,
		date_of_birth Date32,
		city String,
		state String,
		country String,
		tags Array(String),
		notes String,
		created_at DateTime,
		last_activity_at DateTime,
		is_deleted UInt8,
		version DateTime
	) ENGINE = ReplacingMergeTree(version)
	ORDER BY (id);`

	// 2. Contact notes table using ReplacingMergeTree to handle note edits and deletes
	createNotesTable := `
	CREATE TABLE IF NOT EXISTS contact_notes (
		id UUID,
		contact_id UUID,
		note String,
		created_at DateTime,
		updated_at DateTime,
		is_deleted UInt8,
		version DateTime
	) ENGINE = ReplacingMergeTree(version)
	ORDER BY (contact_id, id);`

	// 3. Contact activities timeline table using MergeTree
	// activity_type is String (not Enum) so any module can log any activity
	// dynamically without schema changes
	createActivitiesTable := `
	CREATE TABLE IF NOT EXISTS contact_activities (
		id UUID,
		contact_id UUID,
		activity_type String,
		details String,
		created_at DateTime
	) ENGINE = MergeTree()
	ORDER BY (contact_id, created_at);`

	if err := DB.Exec(ctx, createContactsTable); err != nil {
		panic(fmt.Errorf("failed to create contacts table: %w", err))
	}
	if err := DB.Exec(ctx, createNotesTable); err != nil {
		panic(fmt.Errorf("failed to create contact_notes table: %w", err))
	}
	if err := DB.Exec(ctx, createActivitiesTable); err != nil {
		panic(fmt.Errorf("failed to create contact_activities table: %w", err))
	}

	// Migrate activity_type column from Enum8 to String if it exists as Enum
	// Safe to run — ClickHouse ignores if column is already String
	migrateActivityType := `
	ALTER TABLE contact_activities
	MODIFY COLUMN activity_type String`

	if err := DB.Exec(ctx, migrateActivityType); err != nil {
		// Log warning but don't panic — table may not need migration
		fmt.Printf("Warning: activity_type column migration skipped: %v\n", err)
	} else {
		fmt.Println("contact_activities.activity_type column verified as String (dynamic).")
	}

	fmt.Println("ClickHouse database schema check passed and tables verified.")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}