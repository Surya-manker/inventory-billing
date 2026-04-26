package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/pkg/cache"
	"github.com/yourusername/inventory-billing/pkg/utils"
	"gorm.io/gorm"
)

// AuthTokens is returned on every successful login or token refresh.
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token lifetime in seconds
}

type AuthService interface {
	Register(ctx context.Context, name, email, password string, role domain.Role) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*domain.User, *AuthTokens, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
}

type authService struct {
	userRepo        repository.UserRepository
	tokenStore      *cache.TokenStore
	jwtSecret       string
	accessExpiryHrs int
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenStore *cache.TokenStore,
	jwtSecret string,
	accessExpiryHrs int,
) AuthService {
	return &authService{
		userRepo:        userRepo,
		tokenStore:      tokenStore,
		jwtSecret:       jwtSecret,
		accessExpiryHrs: accessExpiryHrs,
	}
}

func (s *authService) Register(ctx context.Context, name, email, password string, role domain.Role) (*domain.User, error) {
	if len(password) < 8 {
		return nil, domain.ErrWeakPassword
	}

	// validate role — default to staff if unknown value supplied
	if role != domain.RoleAdmin && role != domain.RoleStaff {
		role = domain.RoleStaff
	}

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrEmailTaken
	}

	hashed, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{Name: name, Email: email, Password: hashed, Role: role}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (*domain.User, *AuthTokens, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// mask the "not found" case — caller only gets "invalid credentials"
		return nil, nil, domain.ErrInvalidCredentials
	}
	if !utils.CheckPassword(user.Password, password) {
		return nil, nil, domain.ErrInvalidCredentials
	}

	tokens, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

// RefreshToken validates the refresh token, rotates it (old one is deleted),
// re-fetches the user to pick up any role changes, and issues a fresh pair.
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	payload, err := s.tokenStore.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// rotate: remove the used token before issuing a new one
	_ = s.tokenStore.DeleteRefreshToken(ctx, refreshToken)

	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	return s.issueTokenPair(ctx, user)
}

// Logout invalidates the refresh token. Access tokens remain valid until they naturally expire;
// short access token TTLs (15–60 min) make this acceptable.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	return s.tokenStore.DeleteRefreshToken(ctx, refreshToken)
}

func (s *authService) Me(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (s *authService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return domain.ErrWeakPassword
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return domain.ErrNotFound
	}
	if !utils.CheckPassword(user.Password, oldPassword) {
		return domain.ErrInvalidCredentials
	}

	hashed, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user.Password = hashed
	return s.userRepo.Update(ctx, user)
}

func (s *authService) issueTokenPair(ctx context.Context, user *domain.User) (*AuthTokens, error) {
	accessToken, err := utils.GenerateAccessToken(
		user.ID.String(), user.Email, string(user.Role),
		s.jwtSecret, s.accessExpiryHrs,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenStore.SaveRefreshToken(ctx, cache.RefreshTokenPayload{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
	})
	if err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessExpiryHrs) * 3600,
	}, nil
}
