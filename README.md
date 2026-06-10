# to-do (Golang)

A clean, production-ready RESTful Todo API built with Go, utilizing standard `net/http` library, SQLite for persistence, and JWT for secure authentication.

---

## Features

- **Authentication & Authorization**: Secure signup (`/register`), login (`/login`) with JWT.
- **RESTful Todo Management**: Full CRUD operations for todos:
  - Filtering by status, priority, or search query.
  - Partial updates.
  - Dedicated endpoint to quickly mark a todo as completed.
- **Robust Design Patterns**:
  - **Thread-safe Request Counter**: Utilizing Go mutexes.
  - **Graceful Shutdown**: Handles OS interruption signals (`SIGINT`/`SIGTERM`) and waits for outstanding requests.
  - **Pipeline Concurrency**: Pipeline-based search and filter helper functions.
  - **Timeout Pattern**: Search endpoints protected by timeouts using Go channels.
- **Rate Limiting**: Built-in Token Bucket rate limiter middleware to throttle requests per client IP and prevent spam/brute-force (returns HTTP `429`).
- **Structured Logging**: Using Go's native `log/slog` for JSON-formatted logs to stdout, ideal for log parsing and aggregators.
- **SQLite Concurrency Optimization**: Configured WAL (Write-Ahead Logging) mode, busy timeout, and connection pooling limits to prevent "database is locked" errors under concurrent traffic.
- **Database Migrations**: Automatic SQLite database creation and schema migration on startup.
- **Strict Environment Validation**: Fails fast on startup in production environments (`APP_ENV=production`) if `JWT_SECRET` is unset or remains on default development secrets.

---

## API Documentation

All endpoints (except `/health`, `/register`, and `/login`) require a Bearer token in the `Authorization` header: `Authorization: Bearer <token>`.

| Endpoint | Method | Auth Required | Description |
| :--- | :--- | :---: | :--- |
| `/health` | `GET` | No | Simple health status & server time checks. |
| `/register` | `POST` | No | Creates a new user profile. |
| `/login` | `POST` | No | Authenticates a user and returns a JWT. |
| `/todos` | `GET` | Yes | Get all todos for user (supports `status`, `priority`, `search` query params). |
| `/todos` | `POST` | Yes | Create a new todo. |
| `/todos/{id}` | `GET` | Yes | Retrieve details of a single todo. |
| `/todos/{id}` | `PUT` | Yes | Update an existing todo. |
| `/todos/{id}/complete` | `PATCH` | Yes | Mark a todo as done. |
| `/todos/search` | `GET` | Yes | Search todos with query param `q` (with 5-second timeout logic). |
| `/todos/{id}` | `DELETE` | Yes | Delete a todo. |

A Postman collection is also provided in the repository: [to-do.postman_collection.json](to-do.postman_collection.json).

---

## Local Development Setup

### Prerequisites

This application uses a pure-Go SQLite driver (`modernc.org/sqlite`). This means **no CGO, GCC, or Docker is required** to run or build the project locally on your host system.

You only need:
1. **Go 1.26+**

---

## Running the Application

### Option A: Running Locally (Recommended)

Since the project is CGO-free, you can run the application directly on your machine:

```bash
# Run the application
go run main.go
```

To run on a custom port, you can pass the `-port` flag:
```bash
go run main.go -port 9090
```

### Option B: Using Docker

You can also run the application inside a Docker container:

```bash
# 1. Build the docker image
docker build -t to-do .

# 2. Run the container mapping port 8080
docker run -p 8080:8080 --env-file .env to-do
```

---

## Running Tests

Unit tests are written using Go's built-in `testing` framework. You can run them locally with:

```bash
go test -v ./...
```
