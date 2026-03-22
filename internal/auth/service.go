package auth

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"qwikkle-api/internal/types"
)

type User struct {
	ID           string              `json:"id"`
	QKID         string              `json:"qkId"`
	Email        *string             `json:"email,omitempty"`
	Role         types.UserRole      `json:"role"`
	Status       types.AccountStatus `json:"status"`
	CreatedAt    time.Time           `json:"createdAt"`
	LastLoginAt  *time.Time          `json:"lastLoginAt,omitempty"`
	PasswordHash string              `json:"-"`
}

type Service struct {
	repo      Repository
	jwtSecret string
}

func NewService(repo Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) Signup(ctx context.Context, qkID string, email *string, password string) (*User, string, error) {
	normalizedQKID, err := types.NormalizeQKID(qkID)
	if err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	u, err := s.repo.CreateUser(ctx, normalizedQKID, email, string(hash), string(types.UserRoleUser))
	if err != nil {
		return nil, "", err
	}

	token, err := s.generateToken(u, 24*time.Hour)
	if err != nil {
		return nil, "", err
	}

	return u, token, nil
}

func (s *Service) Login(ctx context.Context, qkID, password string) (*User, string, error) {
	normalizedQKID, err := types.NormalizeQKID(qkID)
	if err != nil {
		return nil, "", err
	}

	u, err := s.repo.GetUserByQKID(ctx, normalizedQKID)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateToken(u, 24*time.Hour)
	if err != nil {
		return nil, "", err
	}

	return u, token, nil
}

func (s *Service) GenerateAccessToken(u *User, ttl time.Duration) (string, error) {
	return s.generateToken(u, ttl)
}

func (s *Service) generateToken(u *User, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":  u.ID,
		"qkId": u.QKID,
		"role": u.Role,
		"exp":  time.Now().Add(ttl).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
