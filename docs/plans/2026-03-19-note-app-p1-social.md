# Note App P1 Social Features Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add social features to the P0 backend — likes, comments with 2-level replies, public explore feed, and Redis-backed leaderboard.

**Architecture:** Extends the existing monolithic Go/Gin backend. New `likes` and `comments` tables. Redis Sorted Set for leaderboard. SocialService handles visibility-gated access. OptionalAuth middleware for public endpoints. Modifies CheckInService to update leaderboard on new check-ins.

**Tech Stack:** Go 1.25, Gin, pgx, go-redis/v9, PostgreSQL, Redis

**Spec:** `docs/superpowers/specs/2026-03-19-note-app-p1-social-design.md`

---

## File Structure

### New files
```
D:/github/note-app/
├── internal/
│   ├── model/
│   │   ├── like.go                 # Like struct
│   │   └── comment.go              # Comment, CommentWithUser, CreateCommentParams
│   ├── repo/
│   │   ├── like.go                 # LikeRepo: Toggle, Delete, Exists, CountByTarget
│   │   ├── like_test.go
│   │   ├── comment.go              # CommentRepo: Create, ListByTarget, ListReplies, Delete, CountByTarget, GetByID
│   │   ├── comment_test.go
│   │   ├── explore.go              # ExploreRepo: ListPublicNotes, ListPublicPlans
│   │   └── explore_test.go
│   ├── service/
│   │   ├── social.go               # SocialService: Like, Unlike, Comment, DeleteComment, GetComments, GetReplies, visibility check
│   │   └── leaderboard.go          # LeaderboardService: IncrementScore, GetLeaderboard (Redis)
│   ├── handler/
│   │   ├── social.go               # SocialHandler: like/unlike/comment endpoints
│   │   └── explore.go              # ExploreHandler: public feed endpoints
│   └── middleware/
│       └── optional_auth.go        # OptionalAuth: parse token if present, pass through if not
├── migrations/
│   ├── 005_create_likes.up.sql
│   ├── 005_create_likes.down.sql
│   ├── 006_create_comments.up.sql
│   └── 006_create_comments.down.sql
```

### Modified files
```
internal/handler/router.go          # Add Social, Explore handlers + routes
internal/handler/plan.go            # Add Leaderboard endpoint
internal/repo/checkin.go            # Modify Upsert to return is_new via xmax
internal/model/checkin.go           # Add IsNew field to CheckIn
internal/service/checkin.go         # Inject LeaderboardService, call on new check-in
cmd/server/main.go                  # Init Redis, wire new repos/services/handlers
```

---

## Task 1: Migrations (likes + comments tables)

**Files:**
- Create: `D:/github/note-app/migrations/005_create_likes.up.sql`
- Create: `D:/github/note-app/migrations/005_create_likes.down.sql`
- Create: `D:/github/note-app/migrations/006_create_comments.up.sql`
- Create: `D:/github/note-app/migrations/006_create_comments.down.sql`

- [ ] **Step 1: Create likes migration**

`migrations/005_create_likes.up.sql`:
```sql
CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(20) NOT NULL,
    target_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, target_type, target_id)
);

CREATE INDEX idx_likes_target ON likes(target_type, target_id);
```

`migrations/005_create_likes.down.sql`:
```sql
DROP TABLE IF EXISTS likes;
```

- [ ] **Step 2: Create comments migration**

`migrations/006_create_comments.up.sql`:
```sql
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type VARCHAR(20) NOT NULL,
    target_id UUID NOT NULL,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_target ON comments(target_type, target_id, created_at);
CREATE INDEX idx_comments_parent ON comments(parent_id);
```

`migrations/006_create_comments.down.sql`:
```sql
DROP TABLE IF EXISTS comments;
```

- [ ] **Step 3: Run migrations**

```bash
cd D:/github/note-app
migrate -path migrations -database "postgres://noteapp:noteapp@localhost:5432/noteapp?sslmode=disable" up
```

- [ ] **Step 4: Verify tables**

```bash
cd D:/github/note-app && docker compose exec postgres psql -U noteapp -c "\dt"
```
Expected: likes and comments tables present

- [ ] **Step 5: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: add likes and comments migration tables"
```

---

## Task 2: Like + Comment Models

**Files:**
- Create: `D:/github/note-app/internal/model/like.go`
- Create: `D:/github/note-app/internal/model/comment.go`

- [ ] **Step 1: Create like model**

`internal/model/like.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type Like struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	TargetType string    `json:"target_type"`
	TargetID   uuid.UUID `json:"target_id"`
	CreatedAt  time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Create comment model**

`internal/model/comment.go`:
```go
package model

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	TargetType string     `json:"target_type"`
	TargetID   uuid.UUID  `json:"target_id"`
	ParentID   *uuid.UUID `json:"parent_id"`
	Content    string     `json:"content"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CommentWithUser struct {
	ID         uuid.UUID        `json:"id"`
	User       UserBrief        `json:"user"`
	Content    string           `json:"content"`
	ParentID   *uuid.UUID       `json:"parent_id"`
	ReplyCount int              `json:"reply_count,omitempty"`
	Replies    []CommentWithUser `json:"replies,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

type UserBrief struct {
	ID        uuid.UUID `json:"id"`
	Nickname  string    `json:"nickname"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
}

type CreateCommentParams struct {
	Content  string     `json:"content" binding:"required,max=2000"`
	ParentID *uuid.UUID `json:"parent_id"`
}

// TargetTypeFromURL converts URL path values to database values.
// "notes" -> "note", "plans" -> "plan", "checkins" -> "check_in"
func TargetTypeFromURL(urlType string) (string, bool) {
	switch urlType {
	case "notes":
		return "note", true
	case "plans":
		return "plan", true
	case "checkins":
		return "check_in", true
	default:
		return "", false
	}
}
```

- [ ] **Step 3: Verify build**

```bash
cd D:/github/note-app && go build ./...
```

- [ ] **Step 4: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: add like and comment models with target type mapping"
```

---

## Task 3: Like Repository + Tests

**Files:**
- Create: `D:/github/note-app/internal/repo/like.go`
- Create: `D:/github/note-app/internal/repo/like_test.go`

- [ ] **Step 1: Write like repo tests**

`internal/repo/like_test.go`:
```go
package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestLikeRepo_CreateAndExists(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	likeRepo := NewLikeRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "liker@test.com", Password: "x", Nickname: "Liker",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Likeable"})

	// Create like
	err := likeRepo.Create(ctx, user.ID, "note", note.ID)
	require.NoError(t, err)

	// Check exists
	exists, err := likeRepo.Exists(ctx, user.ID, "note", note.ID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Count
	count, err := likeRepo.CountByTarget(ctx, "note", note.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestLikeRepo_CreateIdempotent(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	likeRepo := NewLikeRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "idempotent@test.com", Password: "x", Nickname: "Idem",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Test"})

	_ = likeRepo.Create(ctx, user.ID, "note", note.ID)
	err := likeRepo.Create(ctx, user.ID, "note", note.ID) // duplicate
	require.NoError(t, err) // should not error

	count, _ := likeRepo.CountByTarget(ctx, "note", note.ID)
	assert.Equal(t, 1, count) // still 1
}

func TestLikeRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	likeRepo := NewLikeRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "unliker@test.com", Password: "x", Nickname: "Unliker",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Unlikeable"})

	_ = likeRepo.Create(ctx, user.ID, "note", note.ID)
	err := likeRepo.Delete(ctx, user.ID, "note", note.ID)
	require.NoError(t, err)

	exists, _ := likeRepo.Exists(ctx, user.ID, "note", note.ID)
	assert.False(t, exists)
}
```

- [ ] **Step 2: Implement like repo**

`internal/repo/like.go`:
```go
package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LikeRepo struct {
	pool *pgxpool.Pool
}

func NewLikeRepo(pool *pgxpool.Pool) *LikeRepo {
	return &LikeRepo{pool: pool}
}

func (r *LikeRepo) Create(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO likes (user_id, target_type, target_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, target_type, target_id) DO NOTHING`,
		userID, targetType, targetID,
	)
	return err
}

func (r *LikeRepo) Delete(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM likes WHERE user_id = $1 AND target_type = $2 AND target_id = $3`,
		userID, targetType, targetID,
	)
	return err
}

func (r *LikeRepo) Exists(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = $1 AND target_type = $2 AND target_id = $3)`,
		userID, targetType, targetID,
	).Scan(&exists)
	return exists, err
}

func (r *LikeRepo) CountByTarget(ctx context.Context, targetType string, targetID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM likes WHERE target_type = $1 AND target_id = $2`,
		targetType, targetID,
	).Scan(&count)
	return count, err
}
```

- [ ] **Step 3: Run tests**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestLikeRepo -count=1
```
Expected: all 3 PASS

- [ ] **Step 4: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: like repo with create, delete, exists, count"
```

---

## Task 4: Comment Repository + Tests

**Files:**
- Create: `D:/github/note-app/internal/repo/comment.go`
- Create: `D:/github/note-app/internal/repo/comment_test.go`

- [ ] **Step 1: Write comment repo tests**

`internal/repo/comment_test.go`:
```go
package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestCommentRepo_CreateAndList(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "commenter@test.com", Password: "x", Nickname: "Commenter",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Commentable"})

	// Create two top-level comments
	c1, err := commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "First!"})
	require.NoError(t, err)
	assert.Equal(t, "First!", c1.Content)
	assert.Nil(t, c1.ParentID)

	_, _ = commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "Second!"})

	// List top-level comments with reply previews
	comments, total, err := commentRepo.ListByTarget(ctx, "note", note.ID, model.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, comments, 2)
}

func TestCommentRepo_ReplyAndPreview(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "replier@test.com", Password: "x", Nickname: "Replier",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Replyable"})

	parent, _ := commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "Parent"})

	// Create 4 replies
	for i := 0; i < 4; i++ {
		_, err := commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{
			Content:  "Reply " + string(rune('A'+i)),
			ParentID: &parent.ID,
		})
		require.NoError(t, err)
	}

	// List should show parent with reply_count=4 and 3 preview replies
	comments, _, _ := commentRepo.ListByTarget(ctx, "note", note.ID, model.PaginationParams{Page: 1, PageSize: 20})
	assert.Equal(t, 1, len(comments)) // only 1 top-level
	assert.Equal(t, 4, comments[0].ReplyCount)
	assert.Len(t, comments[0].Replies, 3) // preview limit

	// ListReplies should return all 4
	replies, total, _ := commentRepo.ListReplies(ctx, parent.ID, model.PaginationParams{Page: 1, PageSize: 20})
	assert.Equal(t, 4, total)
	assert.Len(t, replies, 4)
}

func TestCommentRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "delcomment@test.com", Password: "x", Nickname: "Deleter",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Deletable"})

	c, _ := commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "To delete"})
	err := commentRepo.Delete(ctx, c.ID)
	require.NoError(t, err)

	_, err = commentRepo.GetByID(ctx, c.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCommentRepo_Count(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "counter@test.com", Password: "x", Nickname: "Counter",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Countable"})
	_, _ = commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "One"})
	_, _ = commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "Two"})

	count, err := commentRepo.CountByTarget(ctx, "note", note.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
```

- [ ] **Step 2: Implement comment repo**

`internal/repo/comment.go`:
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

type CommentRepo struct {
	pool *pgxpool.Pool
}

func NewCommentRepo(pool *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{pool: pool}
}

func (r *CommentRepo) Create(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID, params model.CreateCommentParams) (*model.Comment, error) {
	var c model.Comment
	err := r.pool.QueryRow(ctx,
		`INSERT INTO comments (user_id, target_type, target_id, parent_id, content)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, target_type, target_id, parent_id, content, created_at, updated_at`,
		userID, targetType, targetID, params.ParentID, params.Content,
	).Scan(&c.ID, &c.UserID, &c.TargetType, &c.TargetID, &c.ParentID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}

func (r *CommentRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	var c model.Comment
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, target_type, target_id, parent_id, content, created_at, updated_at
		 FROM comments WHERE id = $1`, id,
	).Scan(&c.ID, &c.UserID, &c.TargetType, &c.TargetID, &c.ParentID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

// ListByTarget returns top-level comments (parent_id IS NULL) with user info,
// reply count, and up to 3 earliest reply previews per comment.
func (r *CommentRepo) ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID, params model.PaginationParams) ([]model.CommentWithUser, int, error) {
	params.Normalize()

	// Count top-level comments
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM comments WHERE target_type = $1 AND target_id = $2 AND parent_id IS NULL`,
		targetType, targetID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch top-level comments with user info
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.content, c.parent_id, c.created_at,
		        u.id, u.nickname, u.avatar_url,
		        (SELECT COUNT(*) FROM comments r WHERE r.parent_id = c.id) AS reply_count
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.target_type = $1 AND c.target_id = $2 AND c.parent_id IS NULL
		 ORDER BY c.created_at ASC
		 LIMIT $3 OFFSET $4`,
		targetType, targetID, params.PageSize, params.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []model.CommentWithUser
	for rows.Next() {
		var cw model.CommentWithUser
		if err := rows.Scan(&cw.ID, &cw.Content, &cw.ParentID, &cw.CreatedAt,
			&cw.User.ID, &cw.User.Nickname, &cw.User.AvatarURL, &cw.ReplyCount); err != nil {
			return nil, 0, err
		}
		comments = append(comments, cw)
	}

	// Fetch reply previews (earliest 3) for each top-level comment
	for i := range comments {
		previews, err := r.fetchReplyPreviews(ctx, comments[i].ID, 3)
		if err != nil {
			return nil, 0, err
		}
		comments[i].Replies = previews
	}

	return comments, total, nil
}

func (r *CommentRepo) fetchReplyPreviews(ctx context.Context, parentID uuid.UUID, limit int) ([]model.CommentWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.content, c.parent_id, c.created_at,
		        u.id, u.nickname, u.avatar_url
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.parent_id = $1
		 ORDER BY c.created_at ASC
		 LIMIT $2`,
		parentID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []model.CommentWithUser
	for rows.Next() {
		var cw model.CommentWithUser
		if err := rows.Scan(&cw.ID, &cw.Content, &cw.ParentID, &cw.CreatedAt,
			&cw.User.ID, &cw.User.Nickname, &cw.User.AvatarURL); err != nil {
			return nil, err
		}
		replies = append(replies, cw)
	}
	return replies, nil
}

// ListReplies returns all replies to a given comment, paginated.
func (r *CommentRepo) ListReplies(ctx context.Context, parentID uuid.UUID, params model.PaginationParams) ([]model.CommentWithUser, int, error) {
	params.Normalize()

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM comments WHERE parent_id = $1`, parentID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.content, c.parent_id, c.created_at,
		        u.id, u.nickname, u.avatar_url
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.parent_id = $1
		 ORDER BY c.created_at ASC
		 LIMIT $2 OFFSET $3`,
		parentID, params.PageSize, params.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var replies []model.CommentWithUser
	for rows.Next() {
		var cw model.CommentWithUser
		if err := rows.Scan(&cw.ID, &cw.Content, &cw.ParentID, &cw.CreatedAt,
			&cw.User.ID, &cw.User.Nickname, &cw.User.AvatarURL); err != nil {
			return nil, 0, err
		}
		replies = append(replies, cw)
	}
	return replies, total, nil
}

func (r *CommentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, "DELETE FROM comments WHERE id = $1", id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CommentRepo) CountByTarget(ctx context.Context, targetType string, targetID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM comments WHERE target_type = $1 AND target_id = $2`,
		targetType, targetID,
	).Scan(&count)
	return count, err
}
```

- [ ] **Step 3: Run tests**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestCommentRepo -count=1
```
Expected: all 4 PASS

- [ ] **Step 4: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: comment repo with create, list, replies, delete, count"
```

---

## Task 5: Optional Auth Middleware + Social Service

**Files:**
- Create: `D:/github/note-app/internal/middleware/optional_auth.go`
- Create: `D:/github/note-app/internal/service/social.go`

- [ ] **Step 1: Create optional auth middleware**

`internal/middleware/optional_auth.go`:
```go
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/pkg/utils"
)

// OptionalAuth parses JWT if present. If missing or invalid, continues without user_id.
func OptionalAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		userID, err := utils.ParseJWT(parts[1], jwtSecret)
		if err != nil {
			c.Next() // invalid token = anonymous
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// GetOptionalUserID returns user ID if authenticated, or uuid.Nil if anonymous.
func GetOptionalUserID(c *gin.Context) uuid.UUID {
	val, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	return val.(uuid.UUID)
}
```

- [ ] **Step 2: Create social service**

`internal/service/social.go`:
```go
package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type SocialService struct {
	likeRepo    *repo.LikeRepo
	commentRepo *repo.CommentRepo
	noteRepo    *repo.NoteRepo
	planRepo    *repo.PlanRepo
	checkInRepo *repo.CheckInRepo
}

func NewSocialService(likeRepo *repo.LikeRepo, commentRepo *repo.CommentRepo, noteRepo *repo.NoteRepo, planRepo *repo.PlanRepo, checkInRepo *repo.CheckInRepo) *SocialService {
	return &SocialService{
		likeRepo:    likeRepo,
		commentRepo: commentRepo,
		noteRepo:    noteRepo,
		planRepo:    planRepo,
		checkInRepo: checkInRepo,
	}
}

// checkPublic verifies the target exists and is public. Returns ErrNotFound or ErrNotPublic.
func (s *SocialService) checkPublic(ctx context.Context, targetType string, targetID uuid.UUID) error {
	switch targetType {
	case "note":
		note, err := s.noteRepo.GetByID(ctx, targetID)
		if err != nil {
			return err
		}
		if note.Visibility != "public" {
			return repo.ErrNotFound // hide private content
		}
	case "plan":
		plan, err := s.planRepo.GetByID(ctx, targetID)
		if err != nil {
			return err
		}
		if plan.Visibility != "public" {
			return repo.ErrNotFound
		}
	case "check_in":
		ci, err := s.checkInRepo.GetByID(ctx, targetID)
		if err != nil {
			return err
		}
		plan, err := s.planRepo.GetByID(ctx, ci.PlanID)
		if err != nil {
			return err
		}
		if plan.Visibility != "public" {
			return repo.ErrNotFound
		}
	default:
		return repo.ErrNotFound
	}
	return nil
}

func (s *SocialService) Like(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) error {
	if err := s.checkPublic(ctx, targetType, targetID); err != nil {
		return err
	}
	return s.likeRepo.Create(ctx, userID, targetType, targetID)
}

func (s *SocialService) Unlike(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) error {
	return s.likeRepo.Delete(ctx, userID, targetType, targetID)
}

func (s *SocialService) Comment(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID, params model.CreateCommentParams) (*model.Comment, error) {
	if err := s.checkPublic(ctx, targetType, targetID); err != nil {
		return nil, err
	}

	// Validate 2-level constraint: parent must be top-level
	if params.ParentID != nil {
		parent, err := s.commentRepo.GetByID(ctx, *params.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.ParentID != nil {
			return nil, errors.New("replies to replies are not allowed")
		}
	}

	return s.commentRepo.Create(ctx, userID, targetType, targetID, params)
}

func (s *SocialService) DeleteComment(ctx context.Context, userID uuid.UUID, commentID uuid.UUID) error {
	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.UserID != userID {
		return ErrForbidden
	}
	return s.commentRepo.Delete(ctx, commentID)
}

func (s *SocialService) GetComments(ctx context.Context, targetType string, targetID uuid.UUID, params model.PaginationParams) ([]model.CommentWithUser, int, error) {
	return s.commentRepo.ListByTarget(ctx, targetType, targetID, params)
}

func (s *SocialService) GetReplies(ctx context.Context, commentID uuid.UUID, params model.PaginationParams) ([]model.CommentWithUser, int, error) {
	return s.commentRepo.ListReplies(ctx, commentID, params)
}

// GetSocialCounts returns like_count, comment_count, is_liked for a target.
func (s *SocialService) GetSocialCounts(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) (likeCount, commentCount int, isLiked bool, err error) {
	likeCount, err = s.likeRepo.CountByTarget(ctx, targetType, targetID)
	if err != nil {
		return
	}
	commentCount, err = s.commentRepo.CountByTarget(ctx, targetType, targetID)
	if err != nil {
		return
	}
	if userID != uuid.Nil {
		isLiked, err = s.likeRepo.Exists(ctx, userID, targetType, targetID)
	}
	return
}
```

- [ ] **Step 3: Verify build**

```bash
cd D:/github/note-app && go build ./...
```

- [ ] **Step 4: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: optional auth middleware and social service with visibility checks"
```

---

## Task 6: Social Handler + Routes

**Files:**
- Create: `D:/github/note-app/internal/handler/social.go`
- Modify: `D:/github/note-app/internal/handler/router.go`

- [ ] **Step 1: Create social handler**

`internal/handler/social.go`:
```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type SocialHandler struct {
	socialService *service.SocialService
}

func NewSocialHandler(socialService *service.SocialService) *SocialHandler {
	return &SocialHandler{socialService: socialService}
}

func parseTarget(c *gin.Context) (dbType string, id uuid.UUID, ok bool) {
	urlType := c.Param("target_type")
	dbType, valid := model.TargetTypeFromURL(urlType)
	if !valid {
		RespondBadRequest(c, "invalid target type, must be notes/plans/checkins")
		return "", uuid.Nil, false
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid target id")
		return "", uuid.Nil, false
	}
	return dbType, id, true
}

func (h *SocialHandler) Like(c *gin.Context) {
	targetType, targetID, ok := parseTarget(c)
	if !ok {
		return
	}
	if err := h.socialService.Like(c.Request.Context(), getUserID(c), targetType, targetID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "liked"})
}

func (h *SocialHandler) Unlike(c *gin.Context) {
	targetType, targetID, ok := parseTarget(c)
	if !ok {
		return
	}
	if err := h.socialService.Unlike(c.Request.Context(), getUserID(c), targetType, targetID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "unliked"})
}

func (h *SocialHandler) GetComments(c *gin.Context) {
	targetType, targetID, ok := parseTarget(c)
	if !ok {
		return
	}
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	comments, total, err := h.socialService.GetComments(c.Request.Context(), targetType, targetID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondPaginated(c, comments, total, params)
}

func (h *SocialHandler) CreateComment(c *gin.Context) {
	targetType, targetID, ok := parseTarget(c)
	if !ok {
		return
	}
	var params model.CreateCommentParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	comment, err := h.socialService.Comment(c.Request.Context(), getUserID(c), targetType, targetID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondCreated(c, comment)
}

func (h *SocialHandler) DeleteComment(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid comment id")
		return
	}
	if err := h.socialService.DeleteComment(c.Request.Context(), getUserID(c), commentID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "deleted"})
}

func (h *SocialHandler) GetReplies(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid comment id")
		return
	}
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	replies, total, err := h.socialService.GetReplies(c.Request.Context(), commentID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondPaginated(c, replies, total, params)
}
```

- [ ] **Step 2: Update router — add Social handler and social routes**

Add `Social *SocialHandler` to the `Handlers` struct. Add to protected group:

```go
social := protected.Group("/social")
{
	social.POST("/:target_type/:id/like", h.Social.Like)
	social.DELETE("/:target_type/:id/like", h.Social.Unlike)
	social.GET("/:target_type/:id/comments", h.Social.GetComments)
	social.POST("/:target_type/:id/comments", h.Social.CreateComment)
	social.DELETE("/comments/:id", h.Social.DeleteComment)
	social.GET("/comments/:id/replies", h.Social.GetReplies)
}
```

- [ ] **Step 3: Verify build**

```bash
cd D:/github/note-app && go build ./...
```

- [ ] **Step 4: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: social handler with like, unlike, comment, delete, replies endpoints"
```

---

## Task 7: Explore Repository + Handler

**Files:**
- Create: `D:/github/note-app/internal/repo/explore.go`
- Create: `D:/github/note-app/internal/repo/explore_test.go`
- Create: `D:/github/note-app/internal/handler/explore.go`
- Modify: `D:/github/note-app/internal/handler/router.go`

- [ ] **Step 1: Write explore repo tests**

`internal/repo/explore_test.go`:
```go
package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestExploreRepo_ListPublicNotes(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	exploreRepo := NewExploreRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "explorer@test.com", Password: "x", Nickname: "Explorer",
	}, "$2a$10$dummyhash")

	// Create private + public notes
	_, _ = noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Private", Visibility: "private"})
	pub, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Public"})
	vis := "public"
	_, _ = noteRepo.Update(ctx, pub.ID, model.UpdateNoteParams{Visibility: &vis})

	notes, total, err := exploreRepo.ListPublicNotes(ctx, uuid.Nil, model.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, notes, 1)
	assert.Equal(t, "Public", notes[0]["title"])
}
```

- [ ] **Step 2: Implement explore repo**

`internal/repo/explore.go`:
```go
package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

type ExploreRepo struct {
	pool *pgxpool.Pool
}

func NewExploreRepo(pool *pgxpool.Pool) *ExploreRepo {
	return &ExploreRepo{pool: pool}
}

// ListPublicNotes returns public notes with author info, like/comment counts, is_liked.
func (r *ExploreRepo) ListPublicNotes(ctx context.Context, currentUserID uuid.UUID, params model.PaginationParams) ([]map[string]any, int, error) {
	params.Normalize()

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notes WHERE visibility = 'public' AND is_draft = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	isLikedExpr := "false"
	args := []any{}
	argIdx := 1
	if currentUserID != uuid.Nil {
		isLikedExpr = fmt.Sprintf(
			"EXISTS(SELECT 1 FROM likes WHERE user_id = $%d AND target_type = 'note' AND target_id = n.id)", argIdx)
		args = append(args, currentUserID)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT n.id, n.title, n.content, n.tags, n.created_at,
		       u.id, u.nickname, u.avatar_url,
		       (SELECT COUNT(*) FROM likes WHERE target_type = 'note' AND target_id = n.id) AS like_count,
		       (SELECT COUNT(*) FROM comments WHERE target_type = 'note' AND target_id = n.id) AS comment_count,
		       %s AS is_liked
		FROM notes n
		JOIN users u ON n.user_id = u.id
		WHERE n.visibility = 'public' AND n.is_draft = false
		ORDER BY n.created_at DESC
		LIMIT $%d OFFSET $%d`, isLikedExpr, argIdx, argIdx+1)
	args = append(args, params.PageSize, params.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var (
			noteID                uuid.UUID
			title, content        string
			tags                  json.RawMessage
			createdAt             any
			authorID              uuid.UUID
			nickname              string
			avatarURL             *string
			likeCount, commentCnt int
			isLiked               bool
		)
		if err := rows.Scan(&noteID, &title, &content, &tags, &createdAt,
			&authorID, &nickname, &avatarURL,
			&likeCount, &commentCnt, &isLiked); err != nil {
			return nil, 0, err
		}
		results = append(results, map[string]any{
			"id":            noteID,
			"title":         title,
			"content":       content,
			"tags":          tags,
			"created_at":    createdAt,
			"author":        map[string]any{"id": authorID, "nickname": nickname, "avatar_url": avatarURL},
			"like_count":    likeCount,
			"comment_count": commentCnt,
			"is_liked":      isLiked,
		})
	}
	return results, total, nil
}

// ListPublicPlans returns public plans with author info, like/comment counts, member count.
func (r *ExploreRepo) ListPublicPlans(ctx context.Context, currentUserID uuid.UUID, params model.PaginationParams) ([]map[string]any, int, error) {
	params.Normalize()

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM plans WHERE visibility = 'public'`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	isLikedExpr := "false"
	args := []any{}
	argIdx := 1
	if currentUserID != uuid.Nil {
		isLikedExpr = fmt.Sprintf(
			"EXISTS(SELECT 1 FROM likes WHERE user_id = $%d AND target_type = 'plan' AND target_id = p.id)", argIdx)
		args = append(args, currentUserID)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.title, p.description, p.start_date, p.end_date, p.created_at,
		       u.id, u.nickname, u.avatar_url,
		       (SELECT COUNT(*) FROM likes WHERE target_type = 'plan' AND target_id = p.id) AS like_count,
		       (SELECT COUNT(*) FROM comments WHERE target_type = 'plan' AND target_id = p.id) AS comment_count,
		       (SELECT COUNT(*) FROM plan_members WHERE plan_id = p.id) AS member_count,
		       %s AS is_liked
		FROM plans p
		JOIN users u ON p.user_id = u.id
		WHERE p.visibility = 'public'
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d`, isLikedExpr, argIdx, argIdx+1)
	args = append(args, params.PageSize, params.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var (
			planID                            uuid.UUID
			title, description                string
			startDate, endDate                any
			createdAt                         any
			authorID                          uuid.UUID
			nickname                          string
			avatarURL                         *string
			likeCount, commentCnt, memberCnt  int
			isLiked                           bool
		)
		if err := rows.Scan(&planID, &title, &description, &startDate, &endDate, &createdAt,
			&authorID, &nickname, &avatarURL,
			&likeCount, &commentCnt, &memberCnt, &isLiked); err != nil {
			return nil, 0, err
		}
		results = append(results, map[string]any{
			"id":            planID,
			"title":         title,
			"description":   description,
			"start_date":    startDate,
			"end_date":      endDate,
			"created_at":    createdAt,
			"author":        map[string]any{"id": authorID, "nickname": nickname, "avatar_url": avatarURL},
			"like_count":    likeCount,
			"comment_count": commentCnt,
			"member_count":  memberCnt,
			"is_liked":      isLiked,
		})
	}
	return results, total, nil
}
```

- [ ] **Step 3: Create explore handler**

`internal/handler/explore.go`:
```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/middleware"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type ExploreHandler struct {
	exploreRepo *repo.ExploreRepo
}

func NewExploreHandler(exploreRepo *repo.ExploreRepo) *ExploreHandler {
	return &ExploreHandler{exploreRepo: exploreRepo}
}

func (h *ExploreHandler) ListNotes(c *gin.Context) {
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	userID := middleware.GetOptionalUserID(c)
	notes, total, err := h.exploreRepo.ListPublicNotes(c.Request.Context(), userID, params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, notes, total, params)
}

func (h *ExploreHandler) ListPlans(c *gin.Context) {
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	userID := middleware.GetOptionalUserID(c)
	plans, total, err := h.exploreRepo.ListPublicPlans(c.Request.Context(), userID, params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, plans, total, params)
}
```

- [ ] **Step 4: Update router — add explore routes with OptionalAuth**

Add `Explore *ExploreHandler` to the `Handlers` struct. Add OUTSIDE the protected group:

```go
// Public explore (optional auth for is_liked)
explore := v1.Group("/explore", middleware.OptionalAuth(h.JWTSecret))
{
	explore.GET("/notes", h.Explore.ListNotes)
	explore.GET("/plans", h.Explore.ListPlans)
}
```

- [ ] **Step 5: Run tests and verify build**

```bash
cd D:/github/note-app && go test ./internal/repo/ -v -run TestExploreRepo -count=1
go build ./...
```

- [ ] **Step 6: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: explore repo and handler — public notes/plans feed with social counts"
```

---

## Task 8: Leaderboard Service (Redis) + CheckIn Integration

**Files:**
- Create: `D:/github/note-app/internal/service/leaderboard.go`
- Modify: `D:/github/note-app/internal/repo/checkin.go` — Upsert returns is_new via xmax
- Modify: `D:/github/note-app/internal/model/checkin.go` — add IsNew field
- Modify: `D:/github/note-app/internal/service/checkin.go` — inject LeaderboardService
- Modify: `D:/github/note-app/internal/handler/plan.go` — add Leaderboard endpoint

- [ ] **Step 1: Install go-redis**

```bash
cd D:/github/note-app && go get github.com/redis/go-redis/v9
```

- [ ] **Step 2: Create leaderboard service**

`internal/service/leaderboard.go`:
```go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type LeaderboardService struct {
	rdb         *redis.Client
	checkInRepo *repo.CheckInRepo
	userRepo    *repo.UserRepo
}

func NewLeaderboardService(rdb *redis.Client, checkInRepo *repo.CheckInRepo, userRepo *repo.UserRepo) *LeaderboardService {
	return &LeaderboardService{rdb: rdb, checkInRepo: checkInRepo, userRepo: userRepo}
}

func leaderboardKey(planID uuid.UUID) string {
	return fmt.Sprintf("plan:%s:leaderboard", planID.String())
}

// IncrementScore increments the user's check-in count in the leaderboard.
func (s *LeaderboardService) IncrementScore(ctx context.Context, planID, userID uuid.UUID) error {
	return s.rdb.ZIncrBy(ctx, leaderboardKey(planID), 1, userID.String()).Err()
}

// GetLeaderboard returns top N users by check-in count for a plan.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context, planID uuid.UUID, limit int) ([]model.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Get top N from Redis
	results, err := s.rdb.ZRevRangeWithScores(ctx, leaderboardKey(planID), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]model.LeaderboardEntry, 0, len(results))
	for i, z := range results {
		userIDStr, _ := z.Member.(string)
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}

		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			continue
		}

		// Calculate streak from PostgreSQL
		today := time.Now().Format("2006-01-02")
		streak, _ := s.checkInRepo.CurrentStreak(ctx, planID, userID, today)

		entries = append(entries, model.LeaderboardEntry{
			Rank: i + 1,
			User: model.UserBrief{
				ID:        user.ID,
				Nickname:  user.Nickname,
				AvatarURL: user.AvatarURL,
			},
			CheckInCount: int(z.Score),
			Streak:       streak,
		})
	}
	return entries, nil
}
```

- [ ] **Step 3: Add LeaderboardEntry to model**

Add to `internal/model/checkin.go`:
```go
type LeaderboardEntry struct {
	Rank         int       `json:"rank"`
	User         UserBrief `json:"user"`
	CheckInCount int       `json:"check_in_count"`
	Streak       int       `json:"streak"`
}
```

Note: `UserBrief` is defined in `model/comment.go`. This import is fine since they're in the same package.

- [ ] **Step 4: Modify CheckIn repo — Upsert returns is_new**

Update `internal/repo/checkin.go` Upsert method. Change the SQL to use CTE with `xmax = 0`:

```go
func (r *CheckInRepo) Upsert(ctx context.Context, planID, userID uuid.UUID, date string, params model.UpsertCheckInParams) (*model.CheckIn, bool, error) {
	media := params.Media
	if media == nil {
		media = json.RawMessage(`[]`)
	}

	var isNew bool
	row := r.pool.QueryRow(ctx,
		`WITH upsert AS (
			INSERT INTO check_ins (plan_id, user_id, content, media, checked_date)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (plan_id, user_id, checked_date)
			DO UPDATE SET content = EXCLUDED.content, media = EXCLUDED.media, checked_at = NOW()
			RETURNING *, (xmax = 0) AS is_new
		) SELECT id, plan_id, user_id, content, media, checked_date, checked_at, is_new FROM upsert`,
		planID, userID, params.Content, media, date,
	)

	var ci model.CheckIn
	var checkedDate time.Time
	err := row.Scan(&ci.ID, &ci.PlanID, &ci.UserID, &ci.Content, &ci.Media, &checkedDate, &ci.CheckedAt, &isNew)
	if err != nil {
		return nil, false, err
	}
	ci.CheckedDate = checkedDate.Format("2006-01-02")
	return &ci, isNew, nil
}
```

**IMPORTANT**: This changes the Upsert signature from `(*model.CheckIn, error)` to `(*model.CheckIn, bool, error)`. All callers must be updated:
- `internal/service/checkin.go` CheckIn method
- `internal/repo/checkin_test.go` all calls to Upsert

- [ ] **Step 5: Update CheckInService to call LeaderboardService**

Modify `internal/service/checkin.go`:
```go
type CheckInService struct {
	checkInRepo        *repo.CheckInRepo
	planRepo           *repo.PlanRepo
	leaderboardService *LeaderboardService // can be nil if not initialized
}

func NewCheckInService(checkInRepo *repo.CheckInRepo, planRepo *repo.PlanRepo, leaderboardService *LeaderboardService) *CheckInService {
	return &CheckInService{checkInRepo: checkInRepo, planRepo: planRepo, leaderboardService: leaderboardService}
}

func (s *CheckInService) CheckIn(ctx context.Context, userID uuid.UUID, planID uuid.UUID, params model.UpsertCheckInParams) (*model.CheckIn, error) {
	isMember, err := s.planRepo.IsMember(ctx, planID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}

	today := time.Now().Format("2006-01-02")
	ci, isNew, err := s.checkInRepo.Upsert(ctx, planID, userID, today, params)
	if err != nil {
		return nil, err
	}

	// Update leaderboard on new check-in
	if isNew && s.leaderboardService != nil {
		_ = s.leaderboardService.IncrementScore(ctx, planID, userID)
	}

	return ci, nil
}
```

- [ ] **Step 6: Add Leaderboard handler to plan.go**

Add to `internal/handler/plan.go`:
```go
// Add leaderboardService to PlanHandler or create a separate method.
// Simpler: add a LeaderboardHandler or add the service to PlanHandler.

// Option: Add to Handlers struct and route in router.go
```

Create leaderboard as a method on a new field in Handlers. Or simpler — add it to PlanHandler since it's under /plans/:id/leaderboard.

Update `internal/handler/plan.go` — add LeaderboardService field:
```go
type PlanHandler struct {
	planService        *service.PlanService
	leaderboardService *service.LeaderboardService
}

func NewPlanHandler(planService *service.PlanService, leaderboardService *service.LeaderboardService) *PlanHandler {
	return &PlanHandler{planService: planService, leaderboardService: leaderboardService}
}

func (h *PlanHandler) Leaderboard(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit := 50
	if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
		limit = v
	}

	entries, err := h.leaderboardService.GetLeaderboard(c.Request.Context(), planID, limit)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondOK(c, gin.H{"data": entries})
}
```

Add leaderboard route in router.go under plans:
```go
plans.GET("/:id/leaderboard", h.Plan.Leaderboard)
```

- [ ] **Step 7: Update checkin_test.go — fix Upsert signature**

All Upsert calls in tests need the third return value:
```go
// Change: ci, err := checkInRepo.Upsert(...)
// To:     ci, _, err := checkInRepo.Upsert(...)
```

- [ ] **Step 8: Run all tests**

```bash
cd D:/github/note-app && go test ./... -v -count=1
```
Expected: all pass

- [ ] **Step 9: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: leaderboard service (Redis), checkin xmax integration, leaderboard endpoint"
```

---

## Task 9: Wire Everything in main.go

**Files:**
- Modify: `D:/github/note-app/cmd/server/main.go`

- [ ] **Step 1: Update main.go**

```go
package main

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/user/note-app/internal/config"
	"github.com/user/note-app/internal/handler"
	"github.com/user/note-app/internal/repo"
	"github.com/user/note-app/internal/service"
	"github.com/user/note-app/internal/storage"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	minioClient, err := storage.NewMinIOClient(cfg.MinIO)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	pool, err := repo.NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Repos
	userRepo := repo.NewUserRepo(pool)
	noteRepo := repo.NewNoteRepo(pool)
	planRepo := repo.NewPlanRepo(pool)
	checkInRepo := repo.NewCheckInRepo(pool)
	likeRepo := repo.NewLikeRepo(pool)
	commentRepo := repo.NewCommentRepo(pool)
	exploreRepo := repo.NewExploreRepo(pool)

	// Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpireHours)
	noteService := service.NewNoteService(noteRepo)
	planService := service.NewPlanService(planRepo)
	leaderboardService := service.NewLeaderboardService(rdb, checkInRepo, userRepo)
	checkInService := service.NewCheckInService(checkInRepo, planRepo, leaderboardService)
	socialService := service.NewSocialService(likeRepo, commentRepo, noteRepo, planRepo, checkInRepo)

	// Handlers + Router
	handlers := &handler.Handlers{
		Auth:      handler.NewAuthHandler(authService),
		Note:      handler.NewNoteHandler(noteService),
		Plan:      handler.NewPlanHandler(planService, leaderboardService),
		CheckIn:   handler.NewCheckInHandler(checkInService),
		Upload:    handler.NewUploadHandler(minioClient),
		Social:    handler.NewSocialHandler(socialService),
		Explore:   handler.NewExploreHandler(exploreRepo),
		JWTSecret: cfg.JWTSecret,
	}
	r := handler.SetupRouter(handlers)

	log.Printf("Starting server on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

- [ ] **Step 2: Verify full build and tests**

```bash
cd D:/github/note-app && go build ./... && go test ./... -v -count=1
```
Expected: build clean, all tests pass

- [ ] **Step 3: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: wire P1 social features — Redis, social service, explore, leaderboard"
```

---

## Task 10: Social Counts on Individual GET Endpoints

**Files:**
- Modify: `D:/github/note-app/internal/handler/note.go` — inject SocialService, modify Get to return social counts for public notes
- Modify: `D:/github/note-app/internal/handler/plan.go` — modify Get to return social counts for public plans
- Modify: `D:/github/note-app/cmd/server/main.go` — pass SocialService to NoteHandler

- [ ] **Step 1: Update NoteHandler to include social counts**

Modify `NoteHandler` to accept `*service.SocialService`. Update `NewNoteHandler`:
```go
type NoteHandler struct {
	noteService   *service.NoteService
	socialService *service.SocialService
}

func NewNoteHandler(noteService *service.NoteService, socialService *service.SocialService) *NoteHandler {
	return &NoteHandler{noteService: noteService, socialService: socialService}
}
```

Update the `Get` method to append social counts for public notes:
```go
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

	// For public notes, include social counts
	if note.Visibility == "public" && h.socialService != nil {
		likeCount, commentCount, isLiked, _ := h.socialService.GetSocialCounts(
			c.Request.Context(), getUserID(c), "note", noteID)
		RespondOK(c, gin.H{
			"note":          note,
			"like_count":    likeCount,
			"comment_count": commentCount,
			"is_liked":      isLiked,
		})
		return
	}
	RespondOK(c, note)
}
```

- [ ] **Step 2: Update PlanHandler Get similarly**

Add `socialService *service.SocialService` to `PlanHandler`. Update `Get` to include social counts for public plans.

- [ ] **Step 3: Update main.go**

Pass `socialService` to `NewNoteHandler`:
```go
Note: handler.NewNoteHandler(noteService, socialService),
```

- [ ] **Step 4: Verify build and tests**

```bash
cd D:/github/note-app && go build ./... && go test ./... -v -count=1
```

- [ ] **Step 5: Commit**

```bash
cd D:/github/note-app && git add -A && git commit -m "feat: add social counts to individual note/plan GET endpoints"
```

---

## Summary

After completing all 10 tasks, the P1 backend provides:

| Feature | Endpoints |
|---------|-----------|
| Like | `POST/DELETE /social/:type/:id/like` |
| Comment | `GET/POST /social/:type/:id/comments`, `DELETE /social/comments/:id` |
| Replies | `GET /social/comments/:id/replies` |
| Explore | `GET /explore/notes`, `GET /explore/plans` |
| Leaderboard | `GET /plans/:id/leaderboard` |

Key integrations:
- CheckIn now updates Redis leaderboard on new check-ins (xmax=0 detection)
- Explore feed includes like_count, comment_count, is_liked, author info
- OptionalAuth middleware for anonymous explore access
- SocialService enforces visibility-gated access for likes and comments
