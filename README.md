# Contact Management API

An enterprise-grade CRM backend API built with Go, designed for extreme scalability and high-throughput data processing. This API handles complete contact lifecycle management, activity tracking, and asynchronous bulk CSV imports.

## 🚀 Tech Stack

*   **Language:** Go (Golang)
*   **Web Framework:** [Echo v4](https://echo.labstack.com/)
*   **Primary Database:** [ClickHouse](https://clickhouse.com/) (For high-performance analytical queries and timeline storage)
*   **Caching & State:** [Redis](https://redis.io/) (For tracking async background jobs)
*   **Message Broker:** [Kafka](https://kafka.apache.org/) (For decoupling bulk CSV import processing)
*   **API Documentation:** [Swaggo](https://github.com/swaggo/swag) (Interactive Swagger UI)

## ✨ Core Features

*   **Contact Lifecycle:** Create, Read, Update, Soft-Delete, and Restore contacts.
*   **Advanced Listing:** High-performance listing with pagination, sorting, tag-filtering, and full-text search.
*   **Activity Timeline:** Log and retrieve chronological events (emails sent, meetings attended, etc.) for every contact.
*   **Notes Management:** Add, edit, delete, and view historical notes attached to a contact profile.
*   **Tagging System:** Categorize and filter contacts efficiently using tags (e.g., "VIP", "Prospect").
*   **Asynchronous Bulk Import:** Upload massive CSV files. The API immediately returns an `import_id`, while a Go background worker processes the rows via Kafka and ClickHouse, storing progress state in Redis.
*   **Smart Partial Updates:** RESTful endpoints intelligently perform partial updates without overwriting zero-values.

## 📂 Project Architecture

```
contact-api/
├── cmd/
│   └── main.go                 # Application entry point, connects infrastructure
├── config/
│   └── config.go               # Initialization of ClickHouse, Redis, and Kafka clients
├── docs/                       # Auto-generated Swagger documentation files
├── handlers/                   # Web layer: Request validation and HTTP responses
│   ├── activity_handler.go
│   ├── contact_handler.go
│   ├── contact_list_handler.go
│   └── import_handler.go
├── models/                     # Data structures and validation rules
│   ├── contact.go
│   └── contact_list.go
├── repository/                 # Data layer: ClickHouse SQL queries and execution
│   └── contact_repo.go
├── routes/                     # Router setup and API versioning
│   └── routes.go
├── services/                   # Business logic and Redis caching layer
│   └── redis_service.go
└── worker/                     # Background jobs
    └── import_consumer.go      # Kafka consumer that processes CSV imports
```

## 🛠️ Setup & Installation

### Prerequisites
*   Go 1.20 or higher
*   ClickHouse running on `localhost:9000`
*   Redis running on `localhost:6379`
*   Kafka running on `localhost:9092`

### Environment Variables
Create a `.env` file in the root directory:
```env
PORT=8081
CLICKHOUSE_URL=localhost:9000
REDIS_URL=localhost:6379
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=contact-imports
```

### Running the Project
1. Install dependencies:
   ```bash
   go mod tidy
   ```
2. Start the API Server:
   ```bash
   go run cmd/main.go
   ```

## 📖 API Documentation (Swagger UI)

This project features a fully interactive documentation dashboard. 
Once the server is running, open your web browser and navigate to:
**👉 `http://localhost:8081/swagger/index.html`**

From this UI, you can view the exact JSON schemas and test requests against the live database.

### Regenerating Docs
If you modify the API or add new endpoints, rebuild the documentation by running:
```bash
swag init -g cmd/main.go -o docs --parseDependency
```

## 🔌 API Endpoints Overview

*   `POST /api/v1/contacts` - Create Contact
*   `GET /api/v1/contacts` - List Contacts (Pagination/Search)
*   `GET /api/v1/contacts/:id` - Get Full Profile (Timeline + Notes)
*   `PUT /api/v1/contacts/:id` - Partial/Full Profile Update
*   `DELETE /api/v1/contacts/:id` - Soft Delete
*   `POST /api/v1/contacts/:id/restore` - Restore Contact
*   `PATCH /api/v1/contacts/:id/tags` - Update Tags
*   `POST /api/v1/contacts/import` - Upload CSV File
*   `GET /api/v1/contacts/import/:import_id` - Check Import Status
*   *(And dedicated routes for Notes and Activities)*
