# NoteApp Backend

A personal growth note-taking app backend built with Go, featuring records, plans with check-ins, social interactions, and growth reports.

## Tech Stack

- **Go 1.25+** / Gin framework
- **PostgreSQL 16** — structured data with JSONB
- **MinIO** — file storage (images, videos, audio)
- **Redis** — leaderboard, caching
- **Docker Compose** — infrastructure orchestration

## Features

### P0 — Core
- User authentication (JWT, email + password)
- Notes CRUD with tags, drafts, and Markdown/Delta JSON content
- Plans CRUD with start/end dates and participant management
- Daily check-ins with upsert (one per plan per day)
- File upload via MinIO presigned URLs
- Check-in calendar and streak calculation

### P1 — Social
- Like/unlike (notes, plans, check-ins)
- Comments with 2-level replies
- Public explore feed with social counts
- Redis-backed leaderboard per plan
- Join public plans
- Optional auth for anonymous explore access

### P2 — Growth
- Growth report generation (monthly, quarterly, yearly)
- Aggregated stats: check-in rates, streaks, trends, top plans

## API Endpoints

```
# Auth
POST   /api/v1/auth/register
POST   /api/v1/auth/login

# Notes
GET    /api/v1/notes
POST   /api/v1/notes
GET    /api/v1/notes/:id
PUT    /api/v1/notes/:id
DELETE /api/v1/notes/:id
PUT    /api/v1/notes/:id/share

# Plans
GET    /api/v1/plans
POST   /api/v1/plans
GET    /api/v1/plans/:id
PUT    /api/v1/plans/:id
DELETE /api/v1/plans/:id
PUT    /api/v1/plans/:id/share
POST   /api/v1/plans/:id/join
GET    /api/v1/plans/:id/members
GET    /api/v1/plans/:id/leaderboard

# Check-ins
POST   /api/v1/plans/:id/checkins
GET    /api/v1/plans/:id/checkins
GET    /api/v1/checkins/calendar

# Social
POST   /api/v1/social/:type/:id/like
DELETE /api/v1/social/:type/:id/like
GET    /api/v1/social/:type/:id/comments
POST   /api/v1/social/:type/:id/comments
DELETE /api/v1/social/comments/:id
GET    /api/v1/social/comments/:id/replies

# Explore
GET    /api/v1/explore/notes
GET    /api/v1/explore/plans

# Growth
GET    /api/v1/growth/reports
POST   /api/v1/growth/generate

# Upload
POST   /api/v1/upload/presign
POST   /api/v1/upload/confirm
```

## Project Structure

```
note-app/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── config/                 # Environment-based config
│   ├── middleware/              # JWT auth, CORS, optional auth
│   ├── model/                  # Data models
│   ├── handler/                # HTTP handlers
│   ├── service/                # Business logic
│   ├── repo/                   # Database access
│   └── storage/                # MinIO client
├── migrations/                 # PostgreSQL migrations (001-007)
├── docker-compose.yaml
├── Dockerfile
└── Makefile
```

## Getting Started

### Option 1: Docker Compose (Recommended)

One command to start everything — database auto-initialized, no manual setup needed.

```bash
# Clone the repository
git clone https://github.com/yingmingchen889-svg/note-app.git
cd note-app

# Build and start all services
docker compose up -d --build

# Verify all services are running
docker compose ps
```

That's it! The server is running at `http://localhost:8080`.

- PostgreSQL table structure is auto-initialized via `scripts/init_db.sql`
- MinIO bucket is auto-created with public read policy
- App waits for all dependencies to be healthy before starting

```bash
# View logs
docker compose logs -f app

# Stop all services
docker compose down

# Stop and remove data (clean start)
docker compose down -v
```

### Option 2: Local Development

For development with hot-reload:

```bash
# Prerequisites: Go 1.22+, Docker, golang-migrate CLI

# Start infrastructure only
docker compose up -d postgres minio redis

# Run database migrations
migrate -path migrations -database "postgres://noteapp:noteapp@localhost:5432/noteapp?sslmode=disable" up

# Start the server (with hot-reload)
go run cmd/server/main.go
```

### Environment Variables

Docker Compose mode uses built-in environment variables. For local development, copy `.env.example` to `.env`:

```env
SERVER_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=noteapp
DB_PASSWORD=noteapp
DB_NAME=noteapp
DB_SSLMODE=disable
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=noteapp
MINIO_USE_SSL=false
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
JWT_SECRET=change-me-in-production
JWT_EXPIRE_HOURS=72
```

### Service Ports

| Service | Port | Description |
|---------|------|-------------|
| App | 8080 | API server |
| PostgreSQL | 5432 | Database |
| MinIO API | 9000 | File storage |
| MinIO Console | 9001 | MinIO web UI |
| Redis | 6379 | Cache |

## Related

- [note-app-flutter](https://github.com/yingmingchen889-svg/note-app-flutter) — Flutter frontend

## License

MIT
