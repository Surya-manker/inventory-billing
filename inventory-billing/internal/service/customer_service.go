package service

import (
	"context"
	"errors"

	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"gorm.io/gorm"
)

type CreateCustomerInput struct {
	Name    string
	Email   string
	Phone   string
	Address string
	Company string
	GSTIN   string
}

type UpdateCustomerInput struct {
	Name    string
	Email   string
	Phone   string
	Address string
	Company string
	GSTIN   string
}

type CustomerService interface {
	Create(ctx context.Context, input CreateCustomerInput) (*domain.Customer, error)
	GetByID(ctx context.Context, id uint) (*domain.Customer, error)
	Update(ctx context.Context, id uint, input UpdateCustomerInput) (*domain.Customer, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, search string, limit, offset int) ([]domain.Customer, int64, error)
}

type customerService struct {
	repo repository.CustomerRepository
}

func NewCustomerService(repo repository.CustomerRepository) CustomerService {
	return &customerService{repo: repo}
}

func (s *customerService) Create(ctx context.Context, input CreateCustomerInput) (*domain.Customer, error) {
	existing, err := s.repo.FindByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrCustomerEmailTaken
	}

	c := &domain.Customer{
		Name: input.Name, Email: input.Email, Phone: input.Phone,
		Address: input.Address, Company: input.Company, GSTIN: input.GSTIN,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *customerService) GetByID(ctx context.Context, id uint) (*domain.Customer, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return c, nil
}

func (s *customerService) Update(ctx context.Context, id uint, input UpdateCustomerInput) (*domain.Customer, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	if c.Email != input.Email {
		existing, err := s.repo.FindByEmail(ctx, input.Email)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, domain.ErrCustomerEmailTaken
		}
	}

	c.Name = input.Name
	c.Email = input.Email
	c.Phone = input.Phone
	c.Address = input.Address
	c.Company = input.Company
	c.GSTIN = input.GSTIN

	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *customerService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *customerService) List(ctx context.Context, search string, limit, offset int) ([]domain.Customer, int64, error) {
	return s.repo.List(ctx, search, limit, offset)
}
