# Note App P0 Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Go backend for P0 — user authentication, notes CRUD, plans CRUD, and check-in with upsert — producing a working API server deployable via Docker Compose.

**Architecture:** Single Go binary using Gin framework. Three-layer structure: handler (HTTP) → service (business logic) → repo (database). PostgreSQL for storage, MinIO for file uploads, Redis for caching. All infrastructure runs via Docker Compose.

**Tech Stack:** Go 1.22+, Gin, pgx (PostgreSQL driver), golang-migrate, MinIO Go SDK, go-redis, bcrypt, golang-jwt, testify, dockertest (integration tests)

**Spec:** `docs/superpowers/specs/2026-03-19-note-app-design.md`

---

## File Structure

```
D:/github/note-app/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point: load config, wire deps, start server
├── internal/
│   ├── config/
│   │   └── config.go                  # Env-based config struct + loader
│   ├── middleware/
│   │   ├── auth.go                    # JWT authentication middleware
│   │   └── cors.go                    # CORS middleware
│   ├── model/
│   │   ├── user.go                    # User struct + CreateUserParams
│   │   ├── note.go                    # Note struct + CreateNoteParams + UpdateNoteParams
│   │   ├── plan.go                    # Plan struct + CreatePlanParams
│   │   ├── plan_member.go             # PlanMember struct
│   │   ├── checkin.go                 # CheckIn struct + UpsertCheckInParams
│   │   └── pagination.go             # PaginationParams + PaginatedResponse
│   ├── handler/
│   │   ├── auth.go                    # POST /auth/register, POST /auth/login
│   │   ├── note.go                    # Notes CRUD endpoints
│   │   ├── plan.go                    # Plans CRUD endpoints
│   │   ├── checkin.go                 # Check-in endpoints
│   │   ├── upload.go                  # Presign + confirm endpoints
│   │   ├── response.go               # Standard JSON response helpers (exported)
│   │   └── router.go                 # Wire all routes
│   ├── service/
│   │   ├── auth.go                    # Register, Login, password hashing
│   │   ├── auth_test.go
│   │   ├── note.go                    # Note business logic
│   │   ├── plan.go                    # Plan business logic
│   │   ├── checkin.go                 # Check-in business logic (upsert, streak calc)
│   ├── repo/
│   │   ├── user.go                    # User DB operations
│   │   ├── user_test.go
│   │   ├── note.go                    # Note DB operations
│   │   ├── note_test.go
│   │   ├── plan.go                    # Plan + PlanMember DB operations
│   │   ├── plan_test.go
│   │   ├── checkin.go                 # Check-in DB operations (upsert)
│   │   ├── checkin_test.go
│   │   └── db.go                      # Connection pool setup
│   └── storage/
│       └── minio.go                   # MinIO client: presign, confirm
├── migrations/
│   ├── 001_create_users.up.sql
│   ├── 001_create_users.down.sql
│   ├── 002_create_notes.up.sql
│   ├── 002_create_notes.down.sql
│   ├── 003_create_plans.up.sql
│   ├── 003_create_plans.down.sql
│   ├── 004_create_checkins.up.sql
│   └── 004_create_checkins.down.sql
├── docker-compose.yaml                # PostgreSQL + MinIO + Redis
├── Dockerfile
├── Makefile                           # Common commands: run, test, migrate, lint
├── .env.example
├── go.mod
└── go.sum
```

---

## Task 1: Project Scaffolding + Docker Compose

**Files:**
- Create: `D:/github/note-app/go.mod`
- Create: `D:/github/note-app/docker-compose.yaml`
- Create: `D:/github/note-app/.env.example`
- Create: `D:/github/note-app/Makefile`
- Create: `D:/github/note-app/internal/config/config.go`
- Create: `D:/github/note-app/cmd/server/main.go`

- [ ] **Step 1: Initialize Go module**

```bash
mkdir -p D:/github/note-app && cd D:/github/note-app
git init
go mod init github.com/user/note-app
```

- [ ] **Step 2: Create Docker Compose for infrastructure**

Create `docker-compose.yaml`:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: noteapp
      POSTGRES_PASSWORD: noteapp
      POSTGRES_DB: noteapp
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - miniodata:/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  pgdata:
  miniodata:
```

- [ ] **Step 3: Create .env.example**

```env
# Server
SERVER_PORT=8080

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=noteapp
DB_PASSWORD=noteapp
DB_NAME=noteapp
DB_SSLMODE=disable

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=noteapp
MINIO_USE_SSL=false

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=change-me-in-production
JWT_EXPIRE_HOURS=72
```

- [ ] **Step 4: Create config loader**

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort    string
	DB            DBConfig
	MinIO         MinIOConfig
	Redis         RedisConfig
	JWTSecret     string
	JWTExpireHours int
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d DBConfig) DSN() string {
	return "postgres://" + d.User + ":" + d.Password + "@" + d.Host + ":" + d.Port + "/" + d.Name + "?sslmode=" + d.SSLMode
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "noteapp"),
			Password: getEnv("DB_PASSWORD", "noteapp"),
			Name:     getEnv("DB_NAME", "noteapp"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("MINIO_BUCKET", "noteapp"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpireHours: getEnvInt("JWT_EXPIRE_HOURS", 72),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
```

- [ ] **Step 5: Create minimal main.go**

Create `cmd/server/main.go`:

```go
package main

import (
	"log"

	"github.com/user/note-app/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting server on port %s", cfg.ServerPort)
}
```

- [ ] **Step 6: Create Makefile**

```makefile
.PHONY: run test migrate-up migrate-down lint infra

infra:
	docker compose up -d

infra-down:
	docker compose down

run:
	go run cmd/server/main.go

test:
	go test ./... -v -count=1

migrate-up:
	migrate -path migrations -database "$(DB_DSN)" up

migrate-down:
	migrate -path migrations -database "$(DB_DSN)" down 1

lint:
	golangci-lint run ./...
```

- [ ] **Step 7: Verify it compiles**

```bash
cd D:/github/note-app && go build ./...
```
Expected: no errors

- [ ] **Step 8: Start infrastructure and verify**

```bash
cd D:/github/note-app && docker compose up -d
docker compose ps
```
Expected: postgres, minio, redis all running

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: project scaffolding with Docker Compose, config, and main entry"
```

---

## Task 2: Database Connection + Migrations

**Files:**
- Create: `D:/github/note-app/internal/repo/db.go`
- Create: `D:/github/note-app/migrations/001_create_users.up.sql`
- Create: `D:/github/note-app/migrations/001_create_users.down.sql`
- Create: `D:/github/note-app/migrations/002_create_notes.up.sql`
- Create: `D:/github/note-app/migrations/002_create_notes.down.sql`
- Create: `D:/github/note-app/migrations/003_create_plans.up.sql`
- Create: `D:/github/note-app/migrations/003_create_plans.down.sql`
- Create: `D:/github/note-app/migrations/004_create_checkins.up.sql`
- Create: `D:/github/note-app/migrations/004_create_checkins.down.sql`

- [ ] **Step 1: Install dependencies**

```bash
cd D:/github/note-app
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
```

- [ ] **Step 2: Create DB connection pool**

Create `internal/repo/db.go`:

```go
package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 3: Create migration 001 — users table**

`migrations/001_create_users.up.sql`:
```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(20),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100) NOT NULL,
    avatar_url VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

`migrations/001_create_users.down.sql`:
```sql
DROP TABLE IF EXISTS users;
```

- [ ] **Step 4: Create migration 002 — notes table**

`migrations/002_create_notes.up.sql`:
```sql
CREATE TYPE visibility AS ENUM ('private', 'public');

CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    media JSONB NOT NULL DEFAULT '[]',
    tags JSONB NOT NULL DEFAULT '[]',
    visibility visibility NOT NULL DEFAULT 'private',
    is_draft BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notes_user_id ON notes(user_id, created_at DESC);
CREATE INDEX idx_notes_visibility ON notes(visibility, created_at DESC) WHERE visibility = 'public';
CREATE INDEX idx_notes_tags ON notes USING GIN(tags);
```

`migrations/002_create_notes.down.sql`:
```sql
DROP TABLE IF EXISTS notes;
DROP TYPE IF EXISTS visibility;
```

- [ ] **Step 5: Create migration 003 — plans + plan_members tables**

`migrations/003_create_plans.up.sql`:
```sql
CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    visibility visibility NOT NULL DEFAULT 'private',
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plans_user_id ON plans(user_id, created_at DESC);

CREATE TYPE plan_role AS ENUM ('owner', 'member');

CREATE TABLE plan_members (
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role plan_role NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plan_id, user_id)
);
```

`migrations/003_create_plans.down.sql`:
```sql
DROP TABLE IF EXISTS plan_members;
DROP TABLE IF EXISTS plans;
DROP TYPE IF EXISTS plan_role;
```

- [ ] **Step 6: Create migration 004 — check_ins table**

`migrations/004_create_checkins.up.sql`:
```sql
CREATE TABLE check_ins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL DEFAULT '',
    media JSONB NOT NULL DEFAULT '[]',
    checked_date DATE NOT NULL DEFAULT CURRENT_DATE,
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(plan_id, user_id, checked_date)
);

CREATE INDEX idx_checkins_plan_user ON check_ins(plan_id, user_id, checked_date DESC);
CREATE INDEX idx_checkins_user_date ON check_ins(user_id, checked_date DESC);
```

`migrations/004_create_checkins.down.sql`:
```sql
DROP TABLE IF EXISTS check_ins;
```

- [ ] **Step 7: Run migrations**

```bash
# Install migrate CLI if needed:
# go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

cd D:/github/note-app
migrate -path migrations -database "postgres://noteapp:noteapp@localhost:5432/noteapp?sslmode=disable" up
```
Expected: `1/u create_users`, `2/u create_notes`, `3/u create_plans`, `4/u create_checkins`

- [ ] **Step 8: Verify tables exist**

```bash
docker compose exec postgres psql -U noteapp -c "\dt"
```
Expected: users, notes, plans, plan_members, check_ins tables listed

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: database connection pool and migration files for all P0 tables"
```

---

## Task 3: Models + Response Helpers + Pagination

**Files:**
- Create: `D:/github/note-app/internal/model/user.go`
- Create: `D:/github/note-app/internal/model/note.go`
- Create: `D:/github/note-app/internal/model/plan.go`
- Create: `D:/github/note-app/internal/model/plan_member.go`
- Create: `D:/github/note-app/internal/model/checkin.go`
- Create: `D:/github/note-app/internal/model/pagination.go`
- Create: `D:/github/note-app/internal/handler/response.go`

- [ ] **Step 1: Create user model**

`internal/model/user.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Phone        *string   `json:"phone,omitempty"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Nickname     string    `json:"nickname"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RegisterParams struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname" binding:"required,min=1,max=100"`
}

type LoginParams struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
```

- [ ] **Step 2: Create note model**

`internal/model/note.go`:
```go
package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"user_id"`
	Title      string          `json:"title"`
	Content    string          `json:"content"`
	Media      json.RawMessage `json:"media"`
	Tags       json.RawMessage `json:"tags"`
	Visibility string          `json:"visibility"`
	IsDraft    bool            `json:"is_draft"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type CreateNoteParams struct {
	Title      string          `json:"title" binding:"required,max=500"`
	Content    string          `json:"content"`
	Media      json.RawMessage `json:"media"`
	Tags       json.RawMessage `json:"tags"`
	Visibility string          `json:"visibility" binding:"omitempty,oneof=private public"`
	IsDraft    bool            `json:"is_draft"`
}

type UpdateNoteParams struct {
	Title      *string          `json:"title" binding:"omitempty,max=500"`
	Content    *string          `json:"content"`
	Media      *json.RawMessage `json:"media"`
	Tags       *json.RawMessage `json:"tags"`
	Visibility *string          `json:"visibility" binding:"omitempty,oneof=private public"`
	IsDraft    *bool            `json:"is_draft"`
}

type NoteListParams struct {
	Tag string `form:"tag"`
	PaginationParams
}
```

- [ ] **Step 3: Create plan model**

`internal/model/plan.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Visibility  string     `json:"visibility"`
	StartDate   string     `json:"start_date"`
	EndDate     *string    `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreatePlanParams struct {
	Title       string  `json:"title" binding:"required,max=500"`
	Description string  `json:"description"`
	Visibility  string  `json:"visibility" binding:"omitempty,oneof=private public"`
	StartDate   string  `json:"start_date" binding:"required"`
	EndDate     *string `json:"end_date"`
}

type UpdatePlanParams struct {
	Title       *string `json:"title" binding:"omitempty,max=500"`
	Description *string `json:"description"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
}
```

- [ ] **Step 4: Create plan_member model**

`internal/model/plan_member.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type PlanMember struct {
	PlanID   uuid.UUID `json:"plan_id"`
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
	Nickname string    `json:"nickname,omitempty"`
}
```

- [ ] **Step 5: Create check-in model**

`internal/model/checkin.go`:
```go
package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CheckIn struct {
	ID          uuid.UUID       `json:"id"`
	PlanID      uuid.UUID       `json:"plan_id"`
	UserID      uuid.UUID       `json:"user_id"`
	Content     string          `json:"content"`
	Media       json.RawMessage `json:"media"`
	CheckedDate string          `json:"checked_date"`
	CheckedAt   time.Time       `json:"checked_at"`
}

type UpsertCheckInParams struct {
	Content string          `json:"content"`
	Media   json.RawMessage `json:"media"`
}

type CalendarEntry struct {
	Date      string `json:"date"`
	PlanID    uuid.UUID `json:"plan_id"`
	PlanTitle string `json:"plan_title"`
}
```

- [ ] **Step 6: Create pagination model**

`internal/model/pagination.go`:
```go
package model

type PaginationParams struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

func (p *PaginationParams) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 || p.PageSize > 100 {
		p.PageSize = 20
	}
}

func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

type PaginatedResponse struct {
	Data     any `json:"data"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
```

- [ ] **Step 7: Create response helpers**

`internal/handler/response.go` — all helpers are **exported** so `middleware/auth.go` can call them:
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/model"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RespondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func RespondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

func RespondPaginated(c *gin.Context, data any, total int, params model.PaginationParams) {
	c.JSON(http.StatusOK, model.PaginatedResponse{
		Data:     data,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	})
}

func RespondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Code: code, Message: message})
}

func RespondBadRequest(c *gin.Context, message string) {
	RespondError(c, http.StatusBadRequest, "INVALID_INPUT", message)
}

func RespondUnauthorized(c *gin.Context) {
	RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

func RespondForbidden(c *gin.Context) {
	RespondError(c, http.StatusForbidden, "FORBIDDEN", "permission denied")
}

func RespondNotFound(c *gin.Context) {
	RespondError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
}

func RespondConflict(c *gin.Context, message string) {
	RespondError(c, http.StatusConflict, "CONFLICT", message)
}

func RespondInternalError(c *gin.Context) {
	RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
```

**Note:** All handler code that calls these helpers must use the exported names (`RespondOK`, `RespondBadRequest`, etc.).

- [ ] **Step 8: Install uuid dependency and verify build**

```bash
cd D:/github/note-app
go get github.com/google/uuid
go get github.com/gin-gonic/gin
go build ./...
```
Expected: no errors

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: add all P0 models, pagination, and response helpers"
```

---

## Task 4: JWT Utilities + Auth Middleware

**Files:**
- Create: `D:/github/note-app/pkg/utils/jwt.go`
- Create: `D:/github/note-app/pkg/utils/jwt_test.go`
- Create: `D:/github/note-app/pkg/utils/password.go`
- Create: `D:/github/note-app/pkg/utils/password_test.go`
- Create: `D:/github/note-app/internal/middleware/auth.go`
- Create: `D:/github/note-app/internal/middleware/cors.go`

- [ ] **Step 1: Write JWT test**

`pkg/utils/jwt_test.go`:
```go
package utils

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseJWT(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	token, err := GenerateJWT(userID, secret, 72)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := ParseJWT(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, parsed)
}

func TestParseJWT_InvalidToken(t *testing.T) {
	_, err := ParseJWT("invalid-token", "secret")
	assert.Error(t, err)
}

func TestParseJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	token, _ := GenerateJWT(userID, "secret1", 72)
	_, err := ParseJWT(token, "secret2")
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd D:/github/note-app && go test ./pkg/utils/ -v -run TestGenerateAndParseJWT
```
Expected: FAIL — function not defined

- [ ] **Step 3: Implement JWT utilities**

`pkg/utils/jwt.go`:
```go
package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateJWT(userID uuid.UUID, secret string, expireHours int) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Duration(expireHours) * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseJWT(tokenStr, secret string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("missing sub claim")
	}
	return uuid.Parse(sub)
}
```

- [ ] **Step 4: Run JWT tests**

```bash
cd D:/github/note-app
go get github.com/golang-jwt/jwt/v5
go get github.com/stretchr/testify
go test ./pkg/utils/ -v -run TestJWT
```
Expected: all 3 tests PASS

- [ ] **Step 5: Write password test**

`pkg/utils/password_test.go`:
```go
package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "mypassword123"
	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEqual(t, password, hash)
	assert.True(t, CheckPassword(password, hash))
}

func TestCheckPassword_Wrong(t *testing.T) {
	hash, _ := HashPassword("correct")
	assert.False(t, CheckPassword("wrong", hash))
}
```

- [ ] **Step 6: Run test to verify it fails**

```bash
cd D:/github/note-app && go test ./pkg/utils/ -v -run TestHash
```
Expected: FAIL

- [ ] **Step 7: Implement password utilities**

`pkg/utils/password.go`:
```go
package utils

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
```

- [ ] **Step 8: Run password tests**

```bash
cd D:/github/note-app
go get golang.org/x/crypto/bcrypt
go test ./pkg/utils/ -v -run TestHash
```
Expected: PASS

- [ ] **Step 9: Create auth middleware**

`internal/middleware/auth.go`:
```go
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/handler"
	"github.com/user/note-app/pkg/utils"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			handler.RespondUnauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			handler.RespondUnauthorized(c)
			c.Abort()
			return
		}

		userID, err := utils.ParseJWT(parts[1], jwtSecret)
		if err != nil {
			handler.RespondUnauthorized(c)
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
```

- [ ] **Step 10: Create CORS middleware**

`internal/middleware/cors.go`:
```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 11: Verify build**

```bash
cd D:/github/note-app && go build ./...
```
Expected: no errors

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "feat: JWT utilities, password hashing, auth and CORS middleware"
```

---

## Task 5: User Repository + Auth Service

**Files:**
- Create: `D:/github/note-app/internal/repo/user.go`
- Create: `D:/github/note-app/internal/repo/user_test.go`
- Create: `D:/github/note-app/internal/service/auth.go`
- Create: `D:/github/note-app/internal/service/auth_test.go`

- [ ] **Step 1: Write user repo test**

`internal/repo/user_test.go`:
```go
package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

// These tests require a running PostgreSQL instance.
// Run: docker compose up -d postgres
// Set DB_DSN env var or use default test DSN.

func TestUserRepo_CreateAndGetByEmail(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	user, err := repo.Create(ctx, model.RegisterParams{
		Email:    "test@example.com",
		Password: "hashedpassword",
		Nickname: "tester",
	}, "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ01234")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "tester", user.Nickname)
	assert.NotEmpty(t, user.ID)

	found, err := repo.GetByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	assert.ErrorIs(t, err, ErrNotFound)
}
```

Note: `testPool(t)` is a helper that connects to a test database. Create a `internal/repo/test_helpers_test.go`:

```go
package repo

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://noteapp:noteapp@localhost:5432/noteapp?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	// Clean test data
	_, _ = pool.Exec(context.Background(), "DELETE FROM check_ins")
	_, _ = pool.Exec(context.Background(), "DELETE FROM plan_members")
	_, _ = pool.Exec(context.Background(), "DELETE FROM plans")
	_, _ = pool.Exec(context.Background(), "DELETE FROM notes")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users")

	return pool
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestUserRepo
```
Expected: FAIL — NewUserRepo not defined

- [ ] **Step 3: Implement user repo**

`internal/repo/user.go`:
```go
package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

var ErrNotFound = errors.New("not found")

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, params model.RegisterParams, passwordHash string) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, nickname)
		 VALUES ($1, $2, $3)
		 RETURNING id, phone, email, password_hash, nickname, avatar_url, created_at, updated_at`,
		params.Email, passwordHash, params.Nickname,
	).Scan(&user.ID, &user.Phone, &user.Email, &user.PasswordHash,
		&user.Nickname, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	return &user, err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, phone, email, password_hash, nickname, avatar_url, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Phone, &user.Email, &user.PasswordHash,
		&user.Nickname, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, phone, email, password_hash, nickname, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Phone, &user.Email, &user.PasswordHash,
		&user.Nickname, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &user, err
}
```

- [ ] **Step 4: Run user repo tests**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestUserRepo
```
Expected: PASS

- [ ] **Step 5: Write auth service test**

`internal/service/auth_test.go`:
```go
package service

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://noteapp:noteapp@localhost:5432/noteapp?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	_, _ = pool.Exec(context.Background(), "DELETE FROM check_ins")
	_, _ = pool.Exec(context.Background(), "DELETE FROM plan_members")
	_, _ = pool.Exec(context.Background(), "DELETE FROM plans")
	_, _ = pool.Exec(context.Background(), "DELETE FROM notes")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users")
	return pool
}

func TestAuthService_RegisterAndLogin(t *testing.T) {
	pool := testPool(t)
	userRepo := repo.NewUserRepo(pool)
	svc := NewAuthService(userRepo, "test-secret", 72)
	ctx := context.Background()

	// Register
	user, token, err := svc.Register(ctx, model.RegisterParams{
		Email:    "auth@test.com",
		Password: "password123",
		Nickname: "Auth Tester",
	})
	require.NoError(t, err)
	assert.Equal(t, "auth@test.com", user.Email)
	assert.NotEmpty(t, token)

	// Login
	user2, token2, err := svc.Login(ctx, model.LoginParams{
		Email:    "auth@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, user.ID, user2.ID)
	assert.NotEmpty(t, token2)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	pool := testPool(t)
	userRepo := repo.NewUserRepo(pool)
	svc := NewAuthService(userRepo, "test-secret", 72)
	ctx := context.Background()

	_, _, _ = svc.Register(ctx, model.RegisterParams{
		Email: "wrong@test.com", Password: "correct", Nickname: "test",
	})

	_, _, err := svc.Login(ctx, model.LoginParams{
		Email: "wrong@test.com", Password: "incorrect",
	})
	assert.Error(t, err)
}
```

- [ ] **Step 6: Run test to verify it fails**

```bash
cd D:/github/note-app && go test ./internal/service/ -v -run TestAuthService
```
Expected: FAIL

- [ ] **Step 7: Implement auth service**

`internal/service/auth.go`:
```go
package service

import (
	"context"
	"errors"

	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
	"github.com/user/note-app/pkg/utils"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AuthService struct {
	userRepo       *repo.UserRepo
	jwtSecret      string
	jwtExpireHours int
}

func NewAuthService(userRepo *repo.UserRepo, jwtSecret string, jwtExpireHours int) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		jwtSecret:      jwtSecret,
		jwtExpireHours: jwtExpireHours,
	}
}

func (s *AuthService) Register(ctx context.Context, params model.RegisterParams) (*model.User, string, error) {
	hash, err := utils.HashPassword(params.Password)
	if err != nil {
		return nil, "", err
	}
	user, err := s.userRepo.Create(ctx, params, hash)
	if err != nil {
		return nil, "", err
	}
	token, err := utils.GenerateJWT(user.ID, s.jwtSecret, s.jwtExpireHours)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, params model.LoginParams) (*model.User, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, params.Email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}
	if !utils.CheckPassword(params.Password, user.PasswordHash) {
		return nil, "", ErrInvalidCredentials
	}
	token, err := utils.GenerateJWT(user.ID, s.jwtSecret, s.jwtExpireHours)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}
```

- [ ] **Step 8: Run auth service tests**

```bash
cd D:/github/note-app && go test ./internal/service/ -v -run TestAuthService
```
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: user repo and auth service with register/login"
```

---

## Task 6: Auth Handler + Router

**Files:**
- Create: `D:/github/note-app/internal/handler/auth.go`
- Create: `D:/github/note-app/internal/handler/auth_test.go`
- Create: `D:/github/note-app/internal/handler/router.go`
- Modify: `D:/github/note-app/cmd/server/main.go`

- [ ] **Step 1: Implement auth handler**

`internal/handler/auth.go`:
```go
package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var params model.RegisterParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	user, token, err := h.authService.Register(c.Request.Context(), params)
	if err != nil {
		if isDuplicateKeyError(err) {
			RespondConflict(c, "email already registered")
			return
		}
		RespondInternalError(c)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":  user,
		"token": token,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var params model.LoginParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	user, token, err := h.authService.Login(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
			return
		}
		RespondInternalError(c)
		return
	}

	RespondOK(c, gin.H{
		"user":  user,
		"token": token,
	})
}

func isDuplicateKeyError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key"))
}
```

- [ ] **Step 2: Create router**

`internal/handler/router.go`:
```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/middleware"
)

type Handlers struct {
	Auth    *AuthHandler
	JWTSecret string
}

func SetupRouter(h *Handlers) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
		}
	}

	// Protected routes will be added in subsequent tasks
	_ = v1.Group("/").Use(middleware.Auth(h.JWTSecret))

	return r
}
```

- [ ] **Step 3: Update main.go to wire everything**

`cmd/server/main.go`:
```go
package main

import (
	"context"
	"log"

	"github.com/user/note-app/internal/config"
	"github.com/user/note-app/internal/handler"
	"github.com/user/note-app/internal/repo"
	"github.com/user/note-app/internal/service"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// Database
	pool, err := repo.NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Repos
	userRepo := repo.NewUserRepo(pool)

	// Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpireHours)

	// Handlers + Router
	handlers := &handler.Handlers{
		Auth:      handler.NewAuthHandler(authService),
		JWTSecret: cfg.JWTSecret,
	}
	r := handler.SetupRouter(handlers)

	log.Printf("Starting server on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

- [ ] **Step 4: Verify build and manual test**

```bash
cd D:/github/note-app && go build ./cmd/server/

# In another terminal, start the server:
# go run cmd/server/main.go

# Test register:
# curl -X POST http://localhost:8080/api/v1/auth/register \
#   -H "Content-Type: application/json" \
#   -d '{"email":"test@test.com","password":"123456","nickname":"tester"}'

# Test login:
# curl -X POST http://localhost:8080/api/v1/auth/login \
#   -H "Content-Type: application/json" \
#   -d '{"email":"test@test.com","password":"123456"}'
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: auth handler, router, and wired main.go — register/login working"
```

---

## Task 7: Note Repository

**Files:**
- Create: `D:/github/note-app/internal/repo/note.go`
- Create: `D:/github/note-app/internal/repo/note_test.go`

- [ ] **Step 1: Write note repo tests**

`internal/repo/note_test.go`:
```go
package repo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestNoteRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "notetest@test.com", Password: "x", Nickname: "Tester",
	}, "$2a$10$dummyhash")

	note, err := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title:   "My First Note",
		Content: "Hello world",
		Tags:    json.RawMessage(`["life"]`),
	})
	require.NoError(t, err)
	assert.Equal(t, "My First Note", note.Title)
	assert.Equal(t, "private", note.Visibility)

	found, err := noteRepo.GetByID(ctx, note.ID)
	require.NoError(t, err)
	assert.Equal(t, note.ID, found.ID)
}

func TestNoteRepo_List_WithTagFilter(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "tagtest@test.com", Password: "x", Nickname: "Tagger",
	}, "$2a$10$dummyhash")

	_, _ = noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title: "Life Note", Tags: json.RawMessage(`["life"]`),
	})
	_, _ = noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title: "Work Note", Tags: json.RawMessage(`["work"]`),
	})

	// List all
	notes, total, err := noteRepo.ListByUser(ctx, user.ID, model.NoteListParams{
		PaginationParams: model.PaginationParams{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, notes, 2)

	// Filter by tag
	notes, total, err = noteRepo.ListByUser(ctx, user.ID, model.NoteListParams{
		Tag:              "life",
		PaginationParams: model.PaginationParams{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, notes, 1)
	assert.Equal(t, "Life Note", notes[0].Title)
}

func TestNoteRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "updatetest@test.com", Password: "x", Nickname: "Updater",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title: "Original",
	})

	newTitle := "Updated"
	updated, err := noteRepo.Update(ctx, note.ID, model.UpdateNoteParams{
		Title: &newTitle,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Title)
}

func TestNoteRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "deltest@test.com", Password: "x", Nickname: "Deleter",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title: "To Delete",
	})

	err := noteRepo.Delete(ctx, note.ID)
	require.NoError(t, err)

	_, err = noteRepo.GetByID(ctx, note.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestNoteRepo
```
Expected: FAIL

- [ ] **Step 3: Implement note repo**

`internal/repo/note.go`:
```go
package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

type NoteRepo struct {
	pool *pgxpool.Pool
}

func NewNoteRepo(pool *pgxpool.Pool) *NoteRepo {
	return &NoteRepo{pool: pool}
}

func (r *NoteRepo) Create(ctx context.Context, userID uuid.UUID, params model.CreateNoteParams) (*model.Note, error) {
	media := params.Media
	if media == nil {
		media = json.RawMessage(`[]`)
	}
	tags := params.Tags
	if tags == nil {
		tags = json.RawMessage(`[]`)
	}
	visibility := params.Visibility
	if visibility == "" {
		visibility = "private"
	}

	var note model.Note
	err := r.pool.QueryRow(ctx,
		`INSERT INTO notes (user_id, title, content, media, tags, visibility, is_draft)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, user_id, title, content, media, tags, visibility, is_draft, created_at, updated_at`,
		userID, params.Title, params.Content, media, tags, visibility, params.IsDraft,
	).Scan(&note.ID, &note.UserID, &note.Title, &note.Content, &note.Media,
		&note.Tags, &note.Visibility, &note.IsDraft, &note.CreatedAt, &note.UpdatedAt)
	return &note, err
}

func (r *NoteRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Note, error) {
	var note model.Note
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, content, media, tags, visibility, is_draft, created_at, updated_at
		 FROM notes WHERE id = $1`,
		id,
	).Scan(&note.ID, &note.UserID, &note.Title, &note.Content, &note.Media,
		&note.Tags, &note.Visibility, &note.IsDraft, &note.CreatedAt, &note.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &note, err
}

func (r *NoteRepo) ListByUser(ctx context.Context, userID uuid.UUID, params model.NoteListParams) ([]model.Note, int, error) {
	params.Normalize()

	where := "WHERE user_id = $1"
	args := []any{userID}
	argIdx := 2

	if params.Tag != "" {
		where += fmt.Sprintf(" AND tags @> $%d", argIdx)
		args = append(args, fmt.Sprintf(`["%s"]`, params.Tag))
		argIdx++
	}

	// Count
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM notes "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch
	query := fmt.Sprintf(
		`SELECT id, user_id, title, content, media, tags, visibility, is_draft, created_at, updated_at
		 FROM notes %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, params.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notes []model.Note
	for rows.Next() {
		var n model.Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Media,
			&n.Tags, &n.Visibility, &n.IsDraft, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, 0, err
		}
		notes = append(notes, n)
	}
	return notes, total, nil
}

func (r *NoteRepo) Update(ctx context.Context, id uuid.UUID, params model.UpdateNoteParams) (*model.Note, error) {
	sets := []string{}
	args := []any{}
	argIdx := 1

	if params.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *params.Title)
		argIdx++
	}
	if params.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", argIdx))
		args = append(args, *params.Content)
		argIdx++
	}
	if params.Media != nil {
		sets = append(sets, fmt.Sprintf("media = $%d", argIdx))
		args = append(args, *params.Media)
		argIdx++
	}
	if params.Tags != nil {
		sets = append(sets, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, *params.Tags)
		argIdx++
	}
	if params.Visibility != nil {
		sets = append(sets, fmt.Sprintf("visibility = $%d", argIdx))
		args = append(args, *params.Visibility)
		argIdx++
	}
	if params.IsDraft != nil {
		sets = append(sets, fmt.Sprintf("is_draft = $%d", argIdx))
		args = append(args, *params.IsDraft)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE notes SET %s WHERE id = $%d
		 RETURNING id, user_id, title, content, media, tags, visibility, is_draft, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx,
	)

	var note model.Note
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&note.ID, &note.UserID, &note.Title, &note.Content, &note.Media,
		&note.Tags, &note.Visibility, &note.IsDraft, &note.CreatedAt, &note.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &note, err
}

func (r *NoteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, "DELETE FROM notes WHERE id = $1", id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *NoteRepo) UpdateVisibility(ctx context.Context, id uuid.UUID, visibility string) (*model.Note, error) {
	v := visibility
	return r.Update(ctx, id, model.UpdateNoteParams{Visibility: &v})
}
```

- [ ] **Step 4: Run note repo tests**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestNoteRepo
```
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: note repository with CRUD, tag filtering, and pagination"
```

---

## Task 8: Note Service + Handler

**Files:**
- Create: `D:/github/note-app/internal/service/note.go`
- Create: `D:/github/note-app/internal/handler/note.go`
- Modify: `D:/github/note-app/internal/handler/router.go`

- [ ] **Step 1: Implement note service**

`internal/service/note.go`:
```go
package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

var ErrForbidden = errors.New("forbidden")

type NoteService struct {
	noteRepo *repo.NoteRepo
}

func NewNoteService(noteRepo *repo.NoteRepo) *NoteService {
	return &NoteService{noteRepo: noteRepo}
}

func (s *NoteService) Create(ctx context.Context, userID uuid.UUID, params model.CreateNoteParams) (*model.Note, error) {
	return s.noteRepo.Create(ctx, userID, params)
}

func (s *NoteService) GetByID(ctx context.Context, userID uuid.UUID, noteID uuid.UUID) (*model.Note, error) {
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if note.Visibility == "private" && note.UserID != userID {
		return nil, ErrForbidden
	}
	return note, nil
}

func (s *NoteService) List(ctx context.Context, userID uuid.UUID, params model.NoteListParams) ([]model.Note, int, error) {
	return s.noteRepo.ListByUser(ctx, userID, params)
}

func (s *NoteService) Update(ctx context.Context, userID uuid.UUID, noteID uuid.UUID, params model.UpdateNoteParams) (*model.Note, error) {
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if note.UserID != userID {
		return nil, ErrForbidden
	}
	return s.noteRepo.Update(ctx, noteID, params)
}

func (s *NoteService) Delete(ctx context.Context, userID uuid.UUID, noteID uuid.UUID) error {
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return err
	}
	if note.UserID != userID {
		return ErrForbidden
	}
	return s.noteRepo.Delete(ctx, noteID)
}

func (s *NoteService) Share(ctx context.Context, userID uuid.UUID, noteID uuid.UUID) (*model.Note, error) {
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}
	if note.UserID != userID {
		return nil, ErrForbidden
	}
	newVis := "public"
	if note.Visibility == "public" {
		newVis = "private"
	}
	return s.noteRepo.UpdateVisibility(ctx, noteID, newVis)
}
```

- [ ] **Step 2: Implement note handler**

`internal/handler/note.go`:
```go
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
	"github.com/user/note-app/internal/service"
)

type NoteHandler struct {
	noteService *service.NoteService
}

func NewNoteHandler(noteService *service.NoteService) *NoteHandler {
	return &NoteHandler{noteService: noteService}
}

func getUserID(c *gin.Context) uuid.UUID {
	return c.MustGet("user_id").(uuid.UUID)
}

func (h *NoteHandler) Create(c *gin.Context) {
	var params model.CreateNoteParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	note, err := h.noteService.Create(c.Request.Context(), getUserID(c), params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondCreated(c, note)
}

func (h *NoteHandler) Get(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid note id")
		return
	}

	note, err := h.noteService.GetByID(c.Request.Context(), getUserID(c), noteID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, note)
}

func (h *NoteHandler) List(c *gin.Context) {
	var params model.NoteListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	notes, total, err := h.noteService.List(c.Request.Context(), getUserID(c), params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, notes, total, params.PaginationParams)
}

func (h *NoteHandler) Update(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid note id")
		return
	}

	var params model.UpdateNoteParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	note, err := h.noteService.Update(c.Request.Context(), getUserID(c), noteID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, note)
}

func (h *NoteHandler) Delete(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid note id")
		return
	}

	if err := h.noteService.Delete(c.Request.Context(), getUserID(c), noteID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "deleted"})
}

func (h *NoteHandler) Share(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid note id")
		return
	}

	note, err := h.noteService.Share(c.Request.Context(), getUserID(c), noteID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, note)
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		RespondNotFound(c)
	case errors.Is(err, service.ErrForbidden):
		RespondForbidden(c)
	default:
		RespondInternalError(c)
	}
}
```

- [ ] **Step 3: Update router to add note routes**

Update `internal/handler/router.go`:
```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/middleware"
)

type Handlers struct {
	Auth      *AuthHandler
	Note      *NoteHandler
	JWTSecret string
}

func SetupRouter(h *Handlers) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS())

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
		}

		protected := v1.Group("").Use(middleware.Auth(h.JWTSecret))
		{
			notes := protected.Group("/notes")
			{
				notes.GET("", h.Note.List)
				notes.POST("", h.Note.Create)
				notes.GET("/:id", h.Note.Get)
				notes.PUT("/:id", h.Note.Update)
				notes.DELETE("/:id", h.Note.Delete)
				notes.PUT("/:id/share", h.Note.Share)
			}
		}
	}

	return r
}
```

- [ ] **Step 4: Update main.go to wire note dependencies**

Add to `cmd/server/main.go`:
```go
// After userRepo
noteRepo := repo.NewNoteRepo(pool)

// After authService
noteService := service.NewNoteService(noteRepo)

// Update handlers
handlers := &handler.Handlers{
	Auth:      handler.NewAuthHandler(authService),
	Note:      handler.NewNoteHandler(noteService),
	JWTSecret: cfg.JWTSecret,
}
```

- [ ] **Step 5: Verify build**

```bash
cd D:/github/note-app && go build ./...
```
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: note service, handler, and routes — full CRUD with tag filter"
```

---

## Task 9: Plan Repository + Service + Handler

**Files:**
- Create: `D:/github/note-app/internal/repo/plan.go`
- Create: `D:/github/note-app/internal/repo/plan_test.go`
- Create: `D:/github/note-app/internal/service/plan.go`
- Create: `D:/github/note-app/internal/handler/plan.go`
- Modify: `D:/github/note-app/internal/handler/router.go`
- Modify: `D:/github/note-app/cmd/server/main.go`

- [ ] **Step 1: Write plan repo test**

`internal/repo/plan_test.go`:
```go
package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestPlanRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "plantest@test.com", Password: "x", Nickname: "Planner",
	}, "$2a$10$dummyhash")

	plan, err := planRepo.Create(ctx, user.ID, model.CreatePlanParams{
		Title:     "Daily Exercise",
		StartDate: "2026-03-19",
	})
	require.NoError(t, err)
	assert.Equal(t, "Daily Exercise", plan.Title)
	assert.Equal(t, "private", plan.Visibility)

	found, err := planRepo.GetByID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, plan.ID, found.ID)
}

func TestPlanRepo_CreateAddsOwnerMember(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "owntest@test.com", Password: "x", Nickname: "Owner",
	}, "$2a$10$dummyhash")

	plan, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{
		Title: "Test Plan", StartDate: "2026-03-19",
	})

	members, err := planRepo.ListMembers(ctx, plan.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "owner", members[0].Role)
}

func TestPlanRepo_ListByUser(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "listplan@test.com", Password: "x", Nickname: "Lister",
	}, "$2a$10$dummyhash")

	_, _ = planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Plan A", StartDate: "2026-03-19"})
	_, _ = planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Plan B", StartDate: "2026-03-19"})

	plans, total, err := planRepo.ListByUser(ctx, user.ID, model.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, plans, 2)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestPlanRepo
```
Expected: FAIL

- [ ] **Step 3: Implement plan repo**

`internal/repo/plan.go`:
```go
package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

type PlanRepo struct {
	pool *pgxpool.Pool
}

func NewPlanRepo(pool *pgxpool.Pool) *PlanRepo {
	return &PlanRepo{pool: pool}
}

func (r *PlanRepo) Create(ctx context.Context, userID uuid.UUID, params model.CreatePlanParams) (*model.Plan, error) {
	visibility := params.Visibility
	if visibility == "" {
		visibility = "private"
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var plan model.Plan
	err = tx.QueryRow(ctx,
		`INSERT INTO plans (user_id, title, description, visibility, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, title, description, visibility, start_date, end_date, created_at, updated_at`,
		userID, params.Title, params.Description, visibility, params.StartDate, params.EndDate,
	).Scan(&plan.ID, &plan.UserID, &plan.Title, &plan.Description, &plan.Visibility,
		&plan.StartDate, &plan.EndDate, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Auto-add creator as owner
	_, err = tx.Exec(ctx,
		`INSERT INTO plan_members (plan_id, user_id, role) VALUES ($1, $2, 'owner')`,
		plan.ID, userID,
	)
	if err != nil {
		return nil, err
	}

	return &plan, tx.Commit(ctx)
}

func (r *PlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Plan, error) {
	var plan model.Plan
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, visibility, start_date, end_date, created_at, updated_at
		 FROM plans WHERE id = $1`, id,
	).Scan(&plan.ID, &plan.UserID, &plan.Title, &plan.Description, &plan.Visibility,
		&plan.StartDate, &plan.EndDate, &plan.CreatedAt, &plan.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &plan, err
}

func (r *PlanRepo) ListByUser(ctx context.Context, userID uuid.UUID, params model.PaginationParams) ([]model.Plan, int, error) {
	params.Normalize()

	// Include both created and joined plans via plan_members
	countQuery := `SELECT COUNT(*) FROM plans p
		JOIN plan_members pm ON p.id = pm.plan_id
		WHERE pm.user_id = $1`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.user_id, p.title, p.description, p.visibility, p.start_date, p.end_date, p.created_at, p.updated_at
		 FROM plans p
		 JOIN plan_members pm ON p.id = pm.plan_id
		 WHERE pm.user_id = $1 ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`,
		userID, params.PageSize, params.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var plans []model.Plan
	for rows.Next() {
		var p model.Plan
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Description, &p.Visibility,
			&p.StartDate, &p.EndDate, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		plans = append(plans, p)
	}
	return plans, total, nil
}

func (r *PlanRepo) Update(ctx context.Context, id uuid.UUID, params model.UpdatePlanParams) (*model.Plan, error) {
	sets := []string{}
	args := []any{}
	argIdx := 1

	if params.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *params.Title)
		argIdx++
	}
	if params.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *params.Description)
		argIdx++
	}
	if params.StartDate != nil {
		sets = append(sets, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *params.StartDate)
		argIdx++
	}
	if params.EndDate != nil {
		sets = append(sets, fmt.Sprintf("end_date = $%d", argIdx))
		args = append(args, *params.EndDate)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE plans SET %s WHERE id = $%d
		 RETURNING id, user_id, title, description, visibility, start_date, end_date, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx,
	)

	var plan model.Plan
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&plan.ID, &plan.UserID, &plan.Title, &plan.Description, &plan.Visibility,
		&plan.StartDate, &plan.EndDate, &plan.CreatedAt, &plan.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &plan, err
}

func (r *PlanRepo) UpdateVisibility(ctx context.Context, id uuid.UUID, visibility string) (*model.Plan, error) {
	var plan model.Plan
	err := r.pool.QueryRow(ctx,
		`UPDATE plans SET visibility = $1, updated_at = NOW() WHERE id = $2
		 RETURNING id, user_id, title, description, visibility, start_date, end_date, created_at, updated_at`,
		visibility, id,
	).Scan(&plan.ID, &plan.UserID, &plan.Title, &plan.Description, &plan.Visibility,
		&plan.StartDate, &plan.EndDate, &plan.CreatedAt, &plan.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &plan, err
}

func (r *PlanRepo) AddMember(ctx context.Context, planID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO plan_members (plan_id, user_id, role) VALUES ($1, $2, 'member')
		 ON CONFLICT (plan_id, user_id) DO NOTHING`,
		planID, userID,
	)
	return err
}

func (r *PlanRepo) ListMembers(ctx context.Context, planID uuid.UUID) ([]model.PlanMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pm.plan_id, pm.user_id, pm.role, pm.joined_at, u.nickname
		 FROM plan_members pm JOIN users u ON pm.user_id = u.id
		 WHERE pm.plan_id = $1 ORDER BY pm.joined_at`,
		planID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.PlanMember
	for rows.Next() {
		var m model.PlanMember
		if err := rows.Scan(&m.PlanID, &m.UserID, &m.Role, &m.JoinedAt, &m.Nickname); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *PlanRepo) IsMember(ctx context.Context, planID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM plan_members WHERE plan_id = $1 AND user_id = $2)`,
		planID, userID,
	).Scan(&exists)
	return exists, err
}
```

- [ ] **Step 4: Run plan repo tests**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestPlanRepo
```
Expected: all PASS

- [ ] **Step 5: Implement plan service**

`internal/service/plan.go`:
```go
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type PlanService struct {
	planRepo *repo.PlanRepo
}

func NewPlanService(planRepo *repo.PlanRepo) *PlanService {
	return &PlanService{planRepo: planRepo}
}

func (s *PlanService) Create(ctx context.Context, userID uuid.UUID, params model.CreatePlanParams) (*model.Plan, error) {
	return s.planRepo.Create(ctx, userID, params)
}

func (s *PlanService) GetByID(ctx context.Context, userID uuid.UUID, planID uuid.UUID) (*model.Plan, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.Visibility == "private" && plan.UserID != userID {
		return nil, ErrForbidden
	}
	return plan, nil
}

func (s *PlanService) List(ctx context.Context, userID uuid.UUID, params model.PaginationParams) ([]model.Plan, int, error) {
	return s.planRepo.ListByUser(ctx, userID, params)
}

func (s *PlanService) Update(ctx context.Context, userID uuid.UUID, planID uuid.UUID, params model.UpdatePlanParams) (*model.Plan, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	return s.planRepo.Update(ctx, planID, params)
}

func (s *PlanService) Share(ctx context.Context, userID uuid.UUID, planID uuid.UUID) (*model.Plan, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	newVis := "public"
	if plan.Visibility == "public" {
		newVis = "private"
	}
	return s.planRepo.UpdateVisibility(ctx, planID, newVis)
}

func (s *PlanService) Join(ctx context.Context, userID uuid.UUID, planID uuid.UUID) error {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return err
	}
	if plan.Visibility != "public" {
		return ErrForbidden
	}
	return s.planRepo.AddMember(ctx, planID, userID)
}

func (s *PlanService) ListMembers(ctx context.Context, planID uuid.UUID) ([]model.PlanMember, error) {
	return s.planRepo.ListMembers(ctx, planID)
}
```

- [ ] **Step 6: Implement plan handler**

`internal/handler/plan.go`:
```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type PlanHandler struct {
	planService *service.PlanService
}

func NewPlanHandler(planService *service.PlanService) *PlanHandler {
	return &PlanHandler{planService: planService}
}

func (h *PlanHandler) Create(c *gin.Context) {
	var params model.CreatePlanParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	plan, err := h.planService.Create(c.Request.Context(), getUserID(c), params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondCreated(c, plan)
}

func (h *PlanHandler) Get(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	plan, err := h.planService.GetByID(c.Request.Context(), getUserID(c), planID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, plan)
}

func (h *PlanHandler) List(c *gin.Context) {
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()
	plans, total, err := h.planService.List(c.Request.Context(), getUserID(c), params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, plans, total, params)
}

func (h *PlanHandler) Update(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	var params model.UpdatePlanParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	plan, err := h.planService.Update(c.Request.Context(), getUserID(c), planID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, plan)
}

func (h *PlanHandler) Share(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	plan, err := h.planService.Share(c.Request.Context(), getUserID(c), planID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, plan)
}

func (h *PlanHandler) Join(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	if err := h.planService.Join(c.Request.Context(), getUserID(c), planID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "joined"})
}

func (h *PlanHandler) Members(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	members, err := h.planService.ListMembers(c.Request.Context(), planID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, members)
}
```

- [ ] **Step 7: Update router to add plan routes**

Add to the `protected` group in `router.go`:
```go
plans := protected.Group("/plans")
{
	plans.GET("", h.Plan.List)
	plans.POST("", h.Plan.Create)
	plans.GET("/:id", h.Plan.Get)
	plans.PUT("/:id", h.Plan.Update)
	plans.PUT("/:id/share", h.Plan.Share)
	plans.POST("/:id/join", h.Plan.Join)
	plans.GET("/:id/members", h.Plan.Members)
}
```

Also add `Plan *PlanHandler` to the `Handlers` struct.

- [ ] **Step 8: Update main.go to wire plan dependencies**

```go
planRepo := repo.NewPlanRepo(pool)
planService := service.NewPlanService(planRepo)

// Add to handlers:
Plan: handler.NewPlanHandler(planService),
```

- [ ] **Step 9: Verify build**

```bash
cd D:/github/note-app && go build ./...
```
Expected: no errors

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "feat: plan repo, service, handler — CRUD with members and join"
```

---

## Task 10: Check-In Repository + Service + Handler

**Files:**
- Create: `D:/github/note-app/internal/repo/checkin.go`
- Create: `D:/github/note-app/internal/repo/checkin_test.go`
- Create: `D:/github/note-app/internal/service/checkin.go`
- Create: `D:/github/note-app/internal/handler/checkin.go`
- Modify: `D:/github/note-app/internal/handler/router.go`
- Modify: `D:/github/note-app/cmd/server/main.go`

- [ ] **Step 1: Write check-in repo test**

`internal/repo/checkin_test.go`:
```go
package repo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestCheckInRepo_Upsert(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	checkInRepo := NewCheckInRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "checkin@test.com", Password: "x", Nickname: "Checker",
	}, "$2a$10$dummyhash")

	plan, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{
		Title: "Test Plan", StartDate: "2026-03-19",
	})

	// First check-in
	ci, err := checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-19", model.UpsertCheckInParams{
		Content: "First check-in",
	})
	require.NoError(t, err)
	assert.Equal(t, "First check-in", ci.Content)

	// Upsert same date — should overwrite
	ci2, err := checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-19", model.UpsertCheckInParams{
		Content: "Updated check-in",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated check-in", ci2.Content)
	assert.Equal(t, ci.ID, ci2.ID) // same record updated
}

func TestCheckInRepo_ListByPlan(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	checkInRepo := NewCheckInRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "listci@test.com", Password: "x", Nickname: "Lister",
	}, "$2a$10$dummyhash")

	plan, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{
		Title: "List Plan", StartDate: "2026-03-01",
	})

	_, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-01", model.UpsertCheckInParams{Content: "Day 1"})
	_, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-02", model.UpsertCheckInParams{Content: "Day 2"})

	checkins, total, err := checkInRepo.ListByPlan(ctx, plan.ID, model.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, checkins, 2)
}

func TestCheckInRepo_Calendar(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	checkInRepo := NewCheckInRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "cal@test.com", Password: "x", Nickname: "Calendar",
	}, "$2a$10$dummyhash")

	plan1, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Plan A", StartDate: "2026-03-01"})
	plan2, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Plan B", StartDate: "2026-03-01"})

	_, _ = checkInRepo.Upsert(ctx, plan1.ID, user.ID, "2026-03-01", model.UpsertCheckInParams{Content: "A1"})
	_, _ = checkInRepo.Upsert(ctx, plan2.ID, user.ID, "2026-03-01", model.UpsertCheckInParams{Content: "B1"})
	_, _ = checkInRepo.Upsert(ctx, plan1.ID, user.ID, "2026-03-02", model.UpsertCheckInParams{Content: "A2"})

	entries, err := checkInRepo.Calendar(ctx, user.ID, "2026-03-01", "2026-03-31")
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestCheckInRepo_Streak(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	checkInRepo := NewCheckInRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "streak@test.com", Password: "x", Nickname: "Streak",
	}, "$2a$10$dummyhash")

	plan, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Streak Plan", StartDate: "2026-03-01"})

	_, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-17", model.UpsertCheckInParams{Content: "d1"})
	_, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-18", model.UpsertCheckInParams{Content: "d2"})
	_, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-19", model.UpsertCheckInParams{Content: "d3"})

	streak, err := checkInRepo.CurrentStreak(ctx, plan.ID, user.ID, "2026-03-19")
	require.NoError(t, err)
	assert.Equal(t, 3, streak)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestCheckInRepo
```
Expected: FAIL

- [ ] **Step 3: Implement check-in repo**

`internal/repo/checkin.go`:
```go
package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

type CheckInRepo struct {
	pool *pgxpool.Pool
}

func NewCheckInRepo(pool *pgxpool.Pool) *CheckInRepo {
	return &CheckInRepo{pool: pool}
}

func (r *CheckInRepo) Upsert(ctx context.Context, planID, userID uuid.UUID, date string, params model.UpsertCheckInParams) (*model.CheckIn, error) {
	media := params.Media
	if media == nil {
		media = json.RawMessage(`[]`)
	}

	var ci model.CheckIn
	err := r.pool.QueryRow(ctx,
		`INSERT INTO check_ins (plan_id, user_id, content, media, checked_date)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (plan_id, user_id, checked_date)
		 DO UPDATE SET content = EXCLUDED.content, media = EXCLUDED.media, checked_at = NOW()
		 RETURNING id, plan_id, user_id, content, media, checked_date, checked_at`,
		planID, userID, params.Content, media, date,
	).Scan(&ci.ID, &ci.PlanID, &ci.UserID, &ci.Content, &ci.Media, &ci.CheckedDate, &ci.CheckedAt)
	return &ci, err
}

func (r *CheckInRepo) ListByPlan(ctx context.Context, planID uuid.UUID, params model.PaginationParams) ([]model.CheckIn, int, error) {
	params.Normalize()

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM check_ins WHERE plan_id = $1", planID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, plan_id, user_id, content, media, checked_date, checked_at
		 FROM check_ins WHERE plan_id = $1 ORDER BY checked_date DESC LIMIT $2 OFFSET $3`,
		planID, params.PageSize, params.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var checkins []model.CheckIn
	for rows.Next() {
		var ci model.CheckIn
		if err := rows.Scan(&ci.ID, &ci.PlanID, &ci.UserID, &ci.Content, &ci.Media,
			&ci.CheckedDate, &ci.CheckedAt); err != nil {
			return nil, 0, err
		}
		checkins = append(checkins, ci)
	}
	return checkins, total, nil
}

func (r *CheckInRepo) Calendar(ctx context.Context, userID uuid.UUID, startDate, endDate string) ([]model.CalendarEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ci.checked_date, ci.plan_id, p.title
		 FROM check_ins ci JOIN plans p ON ci.plan_id = p.id
		 WHERE ci.user_id = $1 AND ci.checked_date >= $2 AND ci.checked_date <= $3
		 ORDER BY ci.checked_date`,
		userID, startDate, endDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.CalendarEntry
	for rows.Next() {
		var e model.CalendarEntry
		if err := rows.Scan(&e.Date, &e.PlanID, &e.PlanTitle); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *CheckInRepo) CurrentStreak(ctx context.Context, planID, userID uuid.UUID, today string) (int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT checked_date FROM check_ins
		 WHERE plan_id = $1 AND user_id = $2 AND checked_date <= $3
		 ORDER BY checked_date DESC`,
		planID, userID, today,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	streak := 0
	var prevTime time.Time
	todayTime, _ := time.Parse("2006-01-02", today)

	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return 0, err
		}
		date, _ := time.Parse("2006-01-02", dateStr)

		if streak == 0 {
			if date.Equal(todayTime) {
				streak = 1
				prevTime = date
				continue
			}
			break
		}

		// Check if this date is exactly 1 day before prevTime
		diff := prevTime.Sub(date).Hours() / 24
		if diff != 1 {
			break
		}
		streak++
		prevTime = date
	}
	return streak, nil
}

// Unused but needed for interface completeness
func (r *CheckInRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.CheckIn, error) {
	var ci model.CheckIn
	err := r.pool.QueryRow(ctx,
		`SELECT id, plan_id, user_id, content, media, checked_date, checked_at
		 FROM check_ins WHERE id = $1`, id,
	).Scan(&ci.ID, &ci.PlanID, &ci.UserID, &ci.Content, &ci.Media, &ci.CheckedDate, &ci.CheckedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &ci, err
}
```

- [ ] **Step 4: Run check-in repo tests**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestCheckInRepo
```
Expected: all PASS

- [ ] **Step 5: Implement check-in service**

`internal/service/checkin.go`:
```go
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type CheckInService struct {
	checkInRepo *repo.CheckInRepo
	planRepo    *repo.PlanRepo
}

func NewCheckInService(checkInRepo *repo.CheckInRepo, planRepo *repo.PlanRepo) *CheckInService {
	return &CheckInService{checkInRepo: checkInRepo, planRepo: planRepo}
}

func (s *CheckInService) CheckIn(ctx context.Context, userID uuid.UUID, planID uuid.UUID, params model.UpsertCheckInParams) (*model.CheckIn, error) {
	// Verify user is a member of the plan
	isMember, err := s.planRepo.IsMember(ctx, planID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}

	today := time.Now().Format("2006-01-02")
	return s.checkInRepo.Upsert(ctx, planID, userID, today, params)
}

func (s *CheckInService) ListByPlan(ctx context.Context, planID uuid.UUID, params model.PaginationParams) ([]model.CheckIn, int, error) {
	return s.checkInRepo.ListByPlan(ctx, planID, params)
}

func (s *CheckInService) Calendar(ctx context.Context, userID uuid.UUID, startDate, endDate string) ([]model.CalendarEntry, error) {
	return s.checkInRepo.Calendar(ctx, userID, startDate, endDate)
}

func (s *CheckInService) Streak(ctx context.Context, planID, userID uuid.UUID) (int, error) {
	today := time.Now().Format("2006-01-02")
	return s.checkInRepo.CurrentStreak(ctx, planID, userID, today)
}
```

- [ ] **Step 6: Implement check-in handler**

`internal/handler/checkin.go`:
```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type CheckInHandler struct {
	checkInService *service.CheckInService
}

func NewCheckInHandler(checkInService *service.CheckInService) *CheckInHandler {
	return &CheckInHandler{checkInService: checkInService}
}

func (h *CheckInHandler) CheckIn(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}

	var params model.UpsertCheckInParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	ci, err := h.checkInService.CheckIn(c.Request.Context(), getUserID(c), planID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondCreated(c, ci)
}

func (h *CheckInHandler) ListByPlan(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}

	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	checkins, total, err := h.checkInService.ListByPlan(c.Request.Context(), planID, params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, checkins, total, params)
}

func (h *CheckInHandler) Calendar(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")
	if startDate == "" || endDate == "" {
		RespondBadRequest(c, "start_date and end_date are required (YYYY-MM-DD)")
		return
	}

	entries, err := h.checkInService.Calendar(c.Request.Context(), getUserID(c), startDate, endDate)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondOK(c, entries)
}
```

- [ ] **Step 7: Update router to add check-in routes**

Add to the `protected` group in `router.go`:
```go
// Check-in routes (nested under plans)
plans.POST("/:id/checkins", h.CheckIn.CheckIn)
plans.GET("/:id/checkins", h.CheckIn.ListByPlan)

// Calendar (top-level under protected)
protected.GET("/checkins/calendar", h.CheckIn.Calendar)
```

Add `CheckIn *CheckInHandler` to the `Handlers` struct.

- [ ] **Step 8: Update main.go to wire check-in dependencies**

```go
checkInRepo := repo.NewCheckInRepo(pool)
checkInService := service.NewCheckInService(checkInRepo, planRepo)

// Add to handlers:
CheckIn: handler.NewCheckInHandler(checkInService),
```

- [ ] **Step 9: Verify build**

```bash
cd D:/github/note-app && go build ./...
```
Expected: no errors

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "feat: check-in repo, service, handler — upsert, calendar, streak"
```

---

## Task 11: MinIO Upload (Presign + Confirm)

**Files:**
- Create: `D:/github/note-app/internal/storage/minio.go`
- Create: `D:/github/note-app/internal/handler/upload.go`
- Modify: `D:/github/note-app/internal/handler/router.go`
- Modify: `D:/github/note-app/cmd/server/main.go`

- [ ] **Step 1: Implement MinIO storage client**

`internal/storage/minio.go`:
```go
package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/user/note-app/internal/config"
)

type MinIOClient struct {
	client *minio.Client
	bucket string
}

func NewMinIOClient(cfg config.MinIOConfig) (*MinIOClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &MinIOClient{client: client, bucket: cfg.Bucket}, nil
}

type PresignResult struct {
	URL       string `json:"url"`
	ObjectKey string `json:"object_key"`
}

func (m *MinIOClient) Presign(ctx context.Context, contentType string) (*PresignResult, error) {
	objectKey := fmt.Sprintf("uploads/%s/%s", time.Now().Format("2006/01/02"), uuid.New().String())

	presignedURL, err := m.client.PresignedPutObject(ctx, m.bucket, objectKey, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("presign: %w", err)
	}

	return &PresignResult{
		URL:       presignedURL.String(),
		ObjectKey: objectKey,
	}, nil
}

func (m *MinIOClient) ObjectURL(objectKey string) string {
	u := &url.URL{
		Scheme: "http",
		Host:   m.client.EndpointURL().Host,
		Path:   fmt.Sprintf("/%s/%s", m.bucket, objectKey),
	}
	return u.String()
}
```

- [ ] **Step 2: Implement upload handler**

`internal/handler/upload.go`:
```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/storage"
)

type UploadHandler struct {
	minioClient *storage.MinIOClient
}

func NewUploadHandler(minioClient *storage.MinIOClient) *UploadHandler {
	return &UploadHandler{minioClient: minioClient}
}

type PresignRequest struct {
	ContentType string `json:"content_type" binding:"required"`
}

func (h *UploadHandler) Presign(c *gin.Context) {
	var req PresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	result, err := h.minioClient.Presign(c.Request.Context(), req.ContentType)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondOK(c, result)
}

type ConfirmRequest struct {
	ObjectKey string `json:"object_key" binding:"required"`
}

func (h *UploadHandler) Confirm(c *gin.Context) {
	var req ConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	objectURL := h.minioClient.ObjectURL(req.ObjectKey)
	RespondOK(c, gin.H{
		"object_key": req.ObjectKey,
		"url":        objectURL,
	})
}
```

- [ ] **Step 3: Update router to add upload routes**

Add to the `protected` group in `router.go`:
```go
upload := protected.Group("/upload")
{
	upload.POST("/presign", h.Upload.Presign)
	upload.POST("/confirm", h.Upload.Confirm)
}
```

Add `Upload *UploadHandler` to the `Handlers` struct.

- [ ] **Step 4: Update main.go to wire MinIO**

```go
// After config load
minioClient, err := storage.NewMinIOClient(cfg.MinIO)
if err != nil {
	log.Fatalf("Failed to connect to MinIO: %v", err)
}

// Add to handlers:
Upload: handler.NewUploadHandler(minioClient),
```

Add import: `"github.com/user/note-app/internal/storage"`

- [ ] **Step 5: Install MinIO dependency and verify build**

```bash
cd D:/github/note-app
go get github.com/minio/minio-go/v7
go build ./...
```
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: MinIO presign upload and confirm endpoints"
```

---

## Task 12: Dockerfile + Final Integration Test

**Files:**
- Create: `D:/github/note-app/Dockerfile`
- Create: `D:/github/note-app/.gitignore`

- [ ] **Step 1: Create Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /server /server

EXPOSE 8080
CMD ["/server"]
```

- [ ] **Step 2: Create .gitignore**

```gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
/server

# Test
*.test
*.out

# IDE
.idea/
.vscode/
*.swp

# Environment
.env

# OS
.DS_Store
Thumbs.db
```

- [ ] **Step 3: Add app service to docker-compose.yaml**

Append to `docker-compose.yaml` services:
```yaml
  app:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    depends_on:
      - postgres
      - minio
      - redis
```

- [ ] **Step 4: Build Docker image**

```bash
cd D:/github/note-app && docker compose build app
```
Expected: build succeeds

- [ ] **Step 5: Run full stack manual smoke test**

```bash
# Start all services
docker compose up -d

# Wait for services
sleep 5

# Run migrations
migrate -path migrations -database "postgres://noteapp:noteapp@localhost:5432/noteapp?sslmode=disable" up

# Register
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke@test.com","password":"123456","nickname":"Smoke Tester"}'

# Login (capture token)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke@test.com","password":"123456"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

# Create note
curl -s -X POST http://localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Test Note","content":"Hello","tags":["life"]}'

# List notes
curl -s http://localhost:8080/api/v1/notes \
  -H "Authorization: Bearer $TOKEN"

# Create plan
curl -s -X POST http://localhost:8080/api/v1/plans \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Daily Exercise","start_date":"2026-03-19"}'

# Check in (use plan ID from above)
# curl -s -X POST http://localhost:8080/api/v1/plans/{PLAN_ID}/checkins \
#   -H "Content-Type: application/json" \
#   -H "Authorization: Bearer $TOKEN" \
#   -d '{"content":"Completed 30 min workout"}'
```

Expected: all requests return 2xx with valid JSON

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: Dockerfile, gitignore, and docker-compose app service — P0 backend complete"
```

---

## Summary

After completing all 12 tasks, the P0 backend provides:

| Feature | Endpoints |
|---------|-----------|
| Auth | `POST /auth/register`, `POST /auth/login` |
| Notes | `GET/POST/PUT/DELETE /notes`, `PUT /notes/:id/share` |
| Plans | `GET/POST/PUT /plans`, `PUT /plans/:id/share`, `POST /plans/:id/join`, `GET /plans/:id/members` |
| Check-ins | `POST /plans/:id/checkins`, `GET /plans/:id/checkins`, `GET /checkins/calendar` |
| Upload | `POST /upload/presign`, `POST /upload/confirm` |

All deployed via Docker Compose with PostgreSQL, MinIO, and Redis.
