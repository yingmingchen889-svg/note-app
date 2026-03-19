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
