package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
)

type CategoryRepository interface {
	Create(ctx context.Context, c *domain.Category) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	FindByName(ctx context.Context, name string) (*domain.Category, error)
	Update(ctx context.Context, c *domain.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]domain.Category, error)
	HasProducts(ctx context.Context, id uuid.UUID) (bool, error)
}

type categoryRepository struct{ db *gorm.DB }

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, c *domain.Category) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *categoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	var c domain.Category
	err := r.db.WithContext(ctx).Preload("Parent").First(&c, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *categoryRepository) FindByName(ctx context.Context, name string) (*domain.Category, error) {
	var c domain.Category
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *categoryRepository) Update(ctx context.Context, c *domain.Category) error {
	return r.db.WithContext(ctx).Model(c).Updates(map[string]any{
		"name":        c.Name,
		"description": c.Description,
		"parent_id":   c.ParentID,
		"is_active":   c.IsActive,
	}).Error
}

func (r *categoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Category{}, "id = ?", id).Error
}

func (r *categoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	var cats []domain.Category
	err := r.db.WithContext(ctx).
		Preload("Parent").
		Where("is_active = true").
		Order("name ASC").
		Find(&cats).Error
	return cats, err
}

func (r *categoryRepository) HasProducts(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Product{}).
		Where("category_id = ?", id).
		Count(&count).Error
	return count > 0, err
}
