package user

import (
	"context"

	"github.com/ocenb/marketplace/internal/models"
	"github.com/ocenb/marketplace/internal/repos/user"
)

type UserServiceInterface interface {
	Create(ctx context.Context, login, passwordHash string) (*models.UserPublic, error)
	GetByLogin(ctx context.Context, login string) (*models.User, error)
	CheckExists(ctx context.Context, login string) (bool, error)
}

type UserService struct {
	userRepo user.UserRepoInterface
}

func New(userRepo user.UserRepoInterface) UserServiceInterface {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) Create(ctx context.Context, login, passwordHash string) (*models.UserPublic, error) {
	user, err := s.userRepo.Create(ctx, login, passwordHash)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) CheckExists(ctx context.Context, login string) (bool, error) {
	exists, err := s.userRepo.CheckExists(ctx, login)
	if err != nil {
		return false, err
	}

	return exists, nil
}
