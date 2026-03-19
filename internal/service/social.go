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

// checkPublic verifies the target exists and is public. Returns repo.ErrNotFound for private content.
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
