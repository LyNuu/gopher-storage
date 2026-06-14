package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"auth-service/internal/model"
	"auth-service/internal/repository"
)

type AuthService struct {
	repo       authRepository
	cache      cacheRepository
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(repo authRepository, cache cacheRepository, jwtSecret []byte,
	accessTTl time.Duration, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		repo:       repo,
		cache:      cache,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTl,
		refreshTTL: refreshTTL,
	}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func (s *AuthService) Create(ctx context.Context, user model.User) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	newID, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	if user.Role == "" {
		user.Role = model.RoleUser
	}
	userDB := model.UserDB{
		ID:           newID,
		Email:        user.Email,
		PasswordHash: string(passwordHash),
		Role:         user.Role,
	}

	err = s.repo.Create(ctx, userDB)
	if err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			return ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCreds
	}

	claims := Claims{
		UserID: user.ID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := tokenObj.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshToken := uuid.New().String()
	err = s.cache.SetRefreshToken(ctx, refreshToken, user.ID.String(), s.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to save refresh token to storage: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, oldRefreshToken string) (*TokenPair, error) {
	userID, err := s.cache.GetUserIDByRefresh(ctx, oldRefreshToken)
	if err != nil {
		return nil, ErrInvalidCreds
	}

	_ = s.cache.DelRefreshToken(ctx, oldRefreshToken)

	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newAccessToken, err := tokenObj.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign new access token: %w", err)
	}
	newRefreshToken := uuid.New().String()

	err = s.cache.SetRefreshToken(ctx, newRefreshToken, userID, s.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to save new refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
