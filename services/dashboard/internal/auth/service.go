package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"code-runner/dashboard/internal/config"
	"code-runner/shared/models"
)

type IAuthService interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
}

type AuthService struct {
	repo IAuthRepository
	cfg  *config.ConfigVar
}

func NewAuthService(repo IAuthRepository, cfg *config.ConfigVar) IAuthService {
	return &AuthService{
		repo: repo,
		cfg:  cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	if _, err := s.repo.FindByEmail(ctx, email); err == nil {
		return nil, ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         models.RoleMember,
		Plan:         models.PlanFree,
	}

	if err := s.repo.Create(ctx, &user); err != nil {
		return nil, err
	}

	token, expiresAt, err := GenerateToken(&user, s.cfg.JWTSecret, s.cfg.TokenTTL)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:      toUserResponse(&user),
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := GenerateToken(user, s.cfg.JWTSecret, s.cfg.TokenTTL)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:      toUserResponse(user),
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *AuthService) GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := toUserResponse(user)
	return &resp, nil
}

func toUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Plan:      user.Plan,
		CreatedAt: user.CreatedAt,
	}
}

func validateEmail(email string) error {
	if email == "" || len(email) > 254 {
		return errors.New("valid email address is required")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(email, ".") {
		return errors.New("valid email address is required")
	}
	return nil
}

func validatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return errors.New("password cannot be blank or whitespace only")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if len([]byte(password)) > 72 {
		return errors.New("password must not exceed 72 bytes")
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("password must contain at least one letter and one number")
	}
	return nil
}

// Keep unused import guard free
var _ = time.Second
