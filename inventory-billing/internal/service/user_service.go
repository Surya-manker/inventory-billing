package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"gorm.io/gorm"
)

type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Update(ctx context.Context, id uuid.UUID, name, email string) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]domain.User, int64, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, name, email string) (*domain.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	// Check uniqueness before hitting the DB constraint so we return a clean
	// domain error instead of a raw MySQL duplicate-key message.
	if existing, err := s.repo.FindByEmail(ctx, email); err == nil && existing.ID != id {
		return nil, domain.ErrEmailTaken
	}

	user.Name = name
	user.Email = email
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *userService) List(ctx context.Context, limit, offset int) ([]domain.User, int64, error) {
	return s.repo.List(ctx, limit, offset)
}
