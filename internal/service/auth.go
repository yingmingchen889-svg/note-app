package service

import (
	"context"
	"errors"

	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
	"github.com/user/note-app/pkg/utils"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrForbidden = errors.New("forbidden")

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
