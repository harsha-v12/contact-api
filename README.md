# Contact Management API

A high-performance, enterprise-grade Contact Management System built with Golang, Echo, ClickHouse, Redis, Kafka, and Docker.

---

## 🚀 Architecture Overview

This project is designed to handle massive amounts of contact data with blazing-fast read/write speeds and background processing for bulk CSV imports.

- **Golang & Echo:** The core HTTP server providing the RESTful API.
- **ClickHouse:** The primary analytical database, storing contacts, notes, and activity logs.
- **Redis:** An in-memory data store used to track the real-time progress percentages of bulk CSV imports.
- **Kafka & Zookeeper:** The message broker handling background jobs (like processing large CSV files) so the main API doesn't freeze.
- **Docker:** Containerized infrastructure to run Kafka and the AKHQ dashboard locally.

---

## 🏗️ Project Creation (Step-by-Step Architecture)

If you are trying to understand how this project was built from scratch, here is the exact chronological file structure:

### 1. Initialization
- `go.mod` (Go modules initialization)
- `.env` (Secure environment variables)

### 2. The Data Layer (Configurations & Database Connections)
- `config/database.go` (ClickHouse connection)
- `config/redis.go` (Redis connection)
- `config/kafka.go` (Kafka producer connection)

### 3. The Models (Data Structures)
- `models/contact.go` (Contact, Notes, and Activity structs)
- `models/import.go` (CSV Import tracking structs)

### 4. The Repository (SQL Queries & Services)
- `repository/contact_repo.go` (CRUD SQL queries)
- `repository/contact_list_repo.go` (Pagination & search SQL queries)
- `services/redis_service.go` (Redis tracking logic)

### 5. The API Handlers (Processing Requests)
- `handlers/contact_handler.go` (Create, Update, Delete contacts)
- `handlers/contact_list_handler.go` (Listing contacts)
- `handlers/activity_handler.go` (Managing contact activities/notes)
- `handlers/import_handler.go` (Uploading CSV files to Kafka)

### 6. The Background Worker (Heavy Lifting)
- `worker/import_consumer.go` (Kafka consumer that reads the CSV and saves to ClickHouse)

### 7. Tying it Together (Routing & Main)
- `middleware/auth.go` (X-API-Key security check)
- `routes/routes.go` (Maps URLs to Handlers)
- `cmd/main.go` (The entry point that boots up the server)

---

## 🛠️ How to Run the Project

### Prerequisites
1. **Golang** (v1.20+)
2. **Docker Desktop**
3. **Postman** (For testing)

### Step 1: Environment Variables
Ensure you have a `.env` file in the root directory with your database credentials:
```env
PORT=8081
API_KEY=a6b33986d7afd06bcdd55fbd22590c7121ab3b3e3e4c8218226dba77d7c5ae0d
CLICKHOUSE_HOST=o9zp3jdtug.ap-south-1.aws.clickhouse.cloud
CLICKHOUSE_PORT=9440
CLICKHOUSE_DB=default
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=your_password
CLICKHOUSE_SECURE=true
REDIS_ADDR=trending-exquisite-turn-73130.db.redis.io:13914
REDIS_PASSWORD=your_password
REDIS_DB=0
REDIS_SECURE=false
KAFKA_BROKER=localhost:9092
KAFKA_TOPIC=contact-import
```

### Step 2: Start Infrastructure
Open your terminal and start Kafka via Docker:
```powershell
docker compose up -d
```

### Step 3: Start the API
Install packages and start the Go server:
```powershell
go mod tidy
go run ./cmd/main.go
```

### Step 4: Test in Postman
Include your API Key in the headers for all requests:
- **Key:** `X-API-Key`
- **Value:** `a6b33986d7afd06bcdd55fbd22590c7121ab3b3e3e4c8218226dba77d7c5ae0d`

*View live Kafka messages at `http://localhost:8082` (AKHQ Dashboard).*
