# To-Do API

A Go-based REST API for managing personal to-do items with SQLite persistence and JWT authentication.

## Features

- User registration and login with JWT-based auth
- CRUD operations for todos
- Todo filtering and search endpoints
- Todo completion and deletion helpers
- Bulk complete and bulk delete actions
- Reorder support for drag-and-drop sorting
- SQLite database with WAL mode
- Request timeout, rate limiting, CORS, logging, and graceful shutdown
- Static frontend served from the `public/` folder

## Project structure

- `main.go` — server setup and route registration
- `config/` — environment configuration loader
- `database/` — SQLite connection and migrations
- `handlers/` — HTTP handlers for auth and todo endpoints
- `middleware/` — JWT auth, CORS, timeout, logging, rate limiting, recovery
- `models/` — request/response models
- `public/` — frontend assets served by the API
- `tests/` — Go tests

## API endpoints

All protected endpoints require a Bearer token in the `Authorization` header:

```http
Authorization: Bearer <token>
```

| Endpoint | Method | Auth | Description |
|:---|:---|:---:|:---|
| `/health` | GET | No | Health check |
| `/register` | POST | No | Register a new user |
| `/login` | POST | No | Login and receive a JWT |
| `/todos` | GET | Yes | List todos |
| `/todos` | POST | Yes | Create a new todo |
| `/todos/search` | GET | Yes | Search todos with timeout protection |
| `/todos/{id}` | GET | Yes | Get one todo |
| `/todos/{id}` | PUT | Yes | Update a todo |
| `/todos/{id}` | DELETE | Yes | Delete a todo |
| `/todos/{id}/complete` | PATCH | Yes | Mark a todo as completed |
| `/todos/reorder` | POST | Yes | Reorder todos using the client order |
| `/todos/bulk-complete` | POST | Yes | Mark multiple todos as complete |
| `/todos/bulk-delete` | POST | Yes | Delete multiple todos |

## Environment variables

The app loads values from `.env` if present. Supported variables:

```env
APP_ENV=development
APP_PORT=8080
DB_PATH=./todo.db
JWT_SECRET=change_this_secret
JWT_EXPIRY_HOURS=24
```

Notes:
- The app reads `APP_PORT`, not `PORT`.
- In production, `JWT_SECRET` should be set to a strong secret and must not be left as the default value.

## Local development

### Prerequisites

- Go 1.26+
- SQLite support via the modernc driver included in the project
- Optional: Docker

### Run directly

```bash
go mod tidy
go run main.go
```

Override the port with:

```bash
go run main.go -port 9090
```

### Run with Docker

```bash
docker build -t to-do .
docker run -p 8080:8080 --env-file .env to-do
```

## Testing

Run the test suite:

```bash
go test ./...
```

## Notes

- The server serves the frontend from `public/` at the root route.
- The app uses SQLite file storage (`./todo.db` by default).
- The backend includes request logging, timeout middleware, rate limiting, recovery middleware, and CORS support.

## Postman

A sample collection is included at `to-do.postman_collection.json`.
