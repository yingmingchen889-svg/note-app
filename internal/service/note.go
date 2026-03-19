package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

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
