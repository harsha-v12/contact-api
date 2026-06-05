# Contact Management API — Codebase Overview

This document explains **why** this project is built the way it is, how the different pieces connect, and how data flows through the entire system.

## 🏗️ The Architecture: Why these tools?

Instead of building a simple, generic API, this project is built like an "enterprise-grade" system capable of handling millions of contacts. Here is why we chose specific technologies:

1. **Golang (Go):** Chosen for its raw speed and ability to handle thousands of requests at the same time using extremely low memory.
2. **ClickHouse:** A standard database like MySQL or PostgreSQL slows down when you search through millions of rows. ClickHouse is a columnar database designed specifically for lightning-fast analytics and massive data sets.
3. **Redis:** When thousands of users ask "What is the progress of my CSV import?", querying ClickHouse over and over would slow it down. Redis is an in-memory database that responds in milliseconds, making it perfect for tracking live progress.
4. **Apache Kafka:** If a user uploads a CSV with 50,000 contacts, parsing and inserting them instantly would freeze the API and crash the server. Kafka acts as a "waiting room" (message queue), allowing a background worker to process the contacts at a safe, steady pace without making the user wait.

---

## 📁 Codebase Structure

The project is organized using a standard Go folder structure:

*   `cmd/main.go` : The entry point. It boots up the server, connects to databases, and starts the Kafka background worker.
*   `config/` : Handles connecting to ClickHouse, Redis, and Kafka.
*   `models/` : Defines the shape of the data (Structs) for Contacts, Notes, and Activities.
*   `routes/` : The switchboard. It maps URLs (like `GET /api/v1/contacts`) to specific handler functions.
*   `handlers/` : The brain of the API. It reads the incoming requests (JSON or CSV), validates them, and decides what to do next.
*   `repository/` : The database layer. This is the **only** place in the code that actually writes SQL queries to ClickHouse.
*   `services/` : Handles external logic, primarily talking to Redis to update progress.
*   `worker/` : The background processor. It listens to Kafka, reads CSV files, and saves them to the database.

---

## ⚙️ Module 1: Core Contact Management

This module handles creating, viewing, updating, and deleting individual contacts.

### The Flow of a "Create Contact" Request
1. **The Request:** A user sends a `POST /api/v1/contacts` with JSON data.
2. **The Handler (`handlers/contact_handler.go`):** Validates the JSON. It checks if the email is valid and ensures required fields are present.
3. **The Database (`repository/contact_repo.go`):** The handler asks the repository to save the contact. The repository runs an `INSERT INTO contacts` SQL query in ClickHouse.
4. **Activity Logging:** Because we want a timeline of everything that happens to a contact, the handler automatically creates a `contact_created` activity log right after saving.

### Soft Deletes
When you delete a contact, we don't actually erase them from ClickHouse. Instead, we insert a new "version" of the contact with `is_deleted = 1`. This is called a **Soft Delete**. It allows you to use the `POST /:id/restore` endpoint to instantly bring them back!

---

## 🚀 Module 2: High-Volume CSV Import

This module is designed to safely import massive amounts of data without crashing the server.

### The Flow of a CSV Import
1. **The Upload (`handlers/import_handler.go`):**
   * The user uploads a CSV file.
   * The API saves the file temporarily to the `./uploads` folder.
   * It creates a job in **Redis** with `0%` progress.
   * It sends a tiny message to **Kafka** that says: *"Hey, there is a new file waiting to be processed!"*
   * It immediately replies `202 Accepted` to the user so they don't have to wait.

2. **The Background Worker (`worker/import_consumer.go`):**
   * The worker (which is constantly listening to Kafka) receives the message.
   * It opens the CSV file from the `./uploads` folder and starts reading it row by row.
   * For every row, it checks if the email already exists in ClickHouse (`CheckDuplicate`).
   * If it's new, it saves it to ClickHouse.
   * Every 100 rows, it updates the live progress in **Redis**.

3. **The Progress Tracker:**
   * The user constantly calls `GET /api/v1/contacts/import/:id`.
   * This API bypasses ClickHouse entirely and looks directly at **Redis**, returning the live `processed_records` and `completion_percentage` instantly.

### Why do it this way?
If we didn't use Kafka and Redis, a user uploading a 100MB CSV file would have to sit staring at a loading screen for 5 minutes. If their internet disconnected during that time, the upload would fail! By using Kafka, we guarantee the file is processed safely in the background, no matter what happens to the user's internet connection.
