package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"gorm.io/gorm"
)

type CreateCategoryInput struct {
	Name        string
	Description string
	ParentID    *uuid.UUID
}

type CategoryService interface {
	Create(ctx context.Context, in CreateCategoryInput) (*domain.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	Update(ctx context.Context, id uuid.UUID, in CreateCategoryInput) (*domain.Category, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]domain.Category, error)
}

type categoryService struct {
	repo repository.CategoryRepository
}

func NewCategoryService(repo repository.CategoryRepository) CategoryService {
	return &categoryService{repo: repo}
}

func (s *categoryService) Create(ctx context.Context, in CreateCategoryInput) (*domain.Category, error) {
	if existing, err := s.repo.FindByName(ctx, in.Name); err == nil && existing != nil {
		return nil, domain.ErrCategoryNameTaken
	}
	c := &domain.Category{
		Name:        in.Name,
		Description: in.Description,
		ParentID:    in.ParentID,
		IsActive:    true,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *categoryService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *categoryService) Update(ctx context.Context, id uuid.UUID, in CreateCategoryInput) (*domain.Category, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if existing, err := s.repo.FindByName(ctx, in.Name); err == nil && existing.ID != id {
		return nil, domain.ErrCategoryNameTaken
	}
	c.Name, c.Description, c.ParentID = in.Name, in.Description, in.ParentID
	return c, s.repo.Update(ctx, c)
}

func (s *categoryService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	has, err := s.repo.HasProducts(ctx, id)
	if err != nil {
		return err
	}
	if has {
		return domain.ErrCategoryHasProducts
	}
	return s.repo.Delete(ctx, id)
}

func (s *categoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
}
