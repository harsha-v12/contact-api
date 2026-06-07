# Step-by-Step Project Creation Guide

If you wanted to build this exact project completely from scratch (an empty folder), here is the exact chronological order of how to create the files and set up the project.

---

## Phase 1: Initialization
This phase sets up the base Golang project.

1. **Initialize the Go Module:**
   Open a terminal in a new empty folder and run:
   ```powershell
   go mod init contact-management
   ```
   *Creates: `go.mod` (tracks your dependencies)*

2. **Create the Environment File:**
   Create a file to hold your database passwords and API keys.
   *Creates: `.env`*

3. **Install the Core Framework:**
   Run the command to install Echo and the godotenv package:
   ```powershell
   go get github.com/labstack/echo/v4
   go get github.com/joho/godotenv
   ```

---

## Phase 2: The Data Layer (Models & Database)
Before writing any APIs, you must define what the data looks like and how to talk to the databases.

4. **Define the Data Models:**
   Create the Go Structs that mirror your database tables.
   *Creates: `models/contact.go`* (Defines the Contact structure)
   *Creates: `models/import.go`* (Defines the CSV Import tracking structure)

5. **Set up Database Connections:**
   Write the code that officially connects Go to ClickHouse, Redis, and Kafka.
   *Creates: `config/database.go`* (ClickHouse)
   *Creates: `config/redis.go`* (Redis)
   *Creates: `config/kafka.go`* (Kafka)

6. **Write Database Queries (Repository):**
   Write the exact SQL queries needed to Save, Update, and Fetch data from ClickHouse.
   *Creates: `repository/contact_repo.go`* (Handles standard CRUD SQL)
   *Creates: `repository/contact_list_repo.go`* (Handles pagination and searching SQL)

7. **Create the Redis Tracker (Services):**
   Write the code that updates the live CSV progress percentages in Redis.
   *Creates: `services/redis_service.go`*

---

## Phase 3: The API Logic (Handlers)
Now that the database is ready, you build the functions that receive HTTP requests from Postman.

8. **Standard API Endpoints:**
   Write the code to handle JSON requests, generate UUIDs, and talk to the Repository.
   *Creates: `handlers/contact_handler.go`* (Create/Update/Delete)
   *Creates: `handlers/contact_list_handler.go`* (Listing/Filtering)
   *Creates: `handlers/activity_handler.go`* (Activity logs)

9. **The CSV Uploader:**
   Write the code that receives the CSV file from Postman and pushes a message to Kafka.
   *Creates: `handlers/import_handler.go`*

---

## Phase 4: The Background Worker
The worker is what actually processes the CSV file behind the scenes.

10. **The Kafka Consumer:**
    Write the infinite loop that listens for Kafka messages, reads the uploaded CSV row-by-row, and saves it using the Repository.
    *Creates: `worker/import_consumer.go`*

---

## Phase 5: Tying It All Together (Routes & Main)
The final step is to connect the Handlers to URLs, and boot up the server.

11. **Security & Middleware:**
    Write the code that checks for the `X-API-Key` before allowing access.
    *Creates: `middleware/auth.go`*

12. **The Switchboard:**
    Map the HTTP URLs (like `POST /api/v1/contacts`) to the specific Handler files you built in Phase 3.
    *Creates: `routes/routes.go`*

13. **The Entry Point:**
    Write the final file that boots up the databases, starts the Kafka worker, starts the Echo router, and opens port 8081.
    *Creates: `cmd/main.go`*

---

## Phase 6: Run It!
14. **Clean up and Start:**
    ```powershell
    go mod tidy
    go run ./cmd/main.go
    ```
