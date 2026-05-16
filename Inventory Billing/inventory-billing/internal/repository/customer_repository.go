package repository

import (
	"context"

	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
)

type CustomerRepository interface {
	Create(ctx context.Context, customer *domain.Customer) error
	FindByID(ctx context.Context, id uint) (*domain.Customer, error)
	FindByEmail(ctx context.Context, email string) (*domain.Customer, error)
	Update(ctx context.Context, customer *domain.Customer) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, search string, limit, offset int) ([]domain.Customer, int64, error)
}

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) Create(ctx context.Context, customer *domain.Customer) error {
	return r.db.WithContext(ctx).Create(customer).Error
}

func (r *customerRepository) FindByID(ctx context.Context, id uint) (*domain.Customer, error) {
	var c domain.Customer
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *customerRepository) FindByEmail(ctx context.Context, email string) (*domain.Customer, error) {
	var c domain.Customer
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *customerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	return r.db.WithContext(ctx).Save(customer).Error
}

func (r *customerRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Customer{}, id).Error
}

func (r *customerRepository) List(ctx context.Context, search string, limit, offset int) ([]domain.Customer, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.Customer{})
	if search != "" {
		pattern := "%" + search + "%"
		// MySQL LIKE is case-insensitive with utf8mb4_general_ci — no ILIKE needed
		q = q.Where("name LIKE ? OR email LIKE ? OR company LIKE ?", pattern, pattern, pattern)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var customers []domain.Customer
	if err := q.Order("name ASC").Limit(limit).Offset(offset).Find(&customers).Error; err != nil {
		return nil, 0, err
	}
	return customers, total, nil
}
