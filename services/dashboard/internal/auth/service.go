package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"code-runner/shared/dto"
	"code-runner/shared/models"
	userRepo "code-runner/dashboard/internal/repository/user"
)

var (
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

type Service struct {
	userRepo  userRepo.Repository
	jwtSecret string
	tokenTTL  time.Duration
}

func NewService(userRepo userRepo.Repository, jwtSecret string, tokenTTL time.Duration) *Service {
	return &Service{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}
}

func (s *Service) Register(req dto.RegisterRequest) (*dto.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || len(req.Password) < 6 {
		return nil, errors.New("valid email and password (minimum 6 characters) required")
	}

	ctx := context.Background()
	if _, err := s.userRepo.FindByEmail(ctx, email); err == nil {
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

	if err := s.userRepo.Create(ctx, &user); err != nil {
		return nil, err
	}

	token, expiresAt, err := GenerateToken(&user, s.jwtSecret, s.tokenTTL)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		User: dto.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Role:      user.Role,
			Plan:      user.Plan,
			CreatedAt: user.CreatedAt,
		},
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) Login(req dto.LoginRequest) (*dto.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	ctx := context.Background()
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, userRepo.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := GenerateToken(user, s.jwtSecret, s.tokenTTL)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		User: dto.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Role:      user.Role,
			Plan:      user.Plan,
			CreatedAt: user.CreatedAt,
		},
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) GetUser(userID uuid.UUID) (*dto.UserResponse, error) {
	ctx := context.Background()
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userRepo.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Plan:      user.Plan,
		CreatedAt: user.CreatedAt,
	}, nil
}
