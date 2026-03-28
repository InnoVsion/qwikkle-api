package auth

import (
	"context"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"qwikkle-api/internal/types"
)

type User struct {
	ID                string              `json:"id"`
	QKID              string              `json:"qkId"`
	Email             *string             `json:"email,omitempty"`
	Role              types.UserRole      `json:"role"`
	Status            types.AccountStatus `json:"status"`
	FirstName         *string             `json:"firstName,omitempty"`
	LastName          *string             `json:"lastName,omitempty"`
	Phone             *string             `json:"phone,omitempty"`
	AvatarURL         *string             `json:"avatarUrl,omitempty"`
	Gender            *string             `json:"gender,omitempty"`
	DateOfBirth       *time.Time          `json:"dateOfBirth,omitempty"`
	Country           *string             `json:"country,omitempty"`
	Interests         []string            `json:"interests"`
	AvatarStorageKey  *string             `json:"avatarStorageKey,omitempty"`
	AvatarDownloadURL *string             `json:"avatarDownloadUrl,omitempty"`
	OrganizationID    *string             `json:"organizationId,omitempty"`
	CreatedAt         time.Time           `json:"createdAt"`
	LastLoginAt       *time.Time          `json:"lastLoginAt,omitempty"`
	PasswordHash      string              `json:"-"`
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

type SignupInput struct {
	QKID              string
	Email             *string
	Password          string
	FirstName         *string
	LastName          *string
	Phone             *string
	AvatarURL         *string
	Gender            *string
	DateOfBirth       *time.Time
	Country           *string
	Interests         []string
	AvatarStorageKey  *string
	AvatarDownloadURL *string
}

func (s *Service) Signup(ctx context.Context, in SignupInput) (*User, string, error) {
	normalizedQKID, err := types.NormalizeQKID(in.QKID)
	if err != nil {
		return nil, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	u, err := s.repo.CreateUser(ctx, CreateUserInput{
		QKID:              normalizedQKID,
		Email:             in.Email,
		PasswordHash:      string(hash),
		Role:              string(types.UserRoleUser),
		Status:            types.AccountStatusActive,
		FirstName:         in.FirstName,
		LastName:          in.LastName,
		Phone:             in.Phone,
		AvatarURL:         in.AvatarURL,
		Gender:            in.Gender,
		DateOfBirth:       in.DateOfBirth,
		Country:           in.Country,
		Interests:         in.Interests,
		AvatarStorageKey:  in.AvatarStorageKey,
		AvatarDownloadURL: in.AvatarDownloadURL,
	})
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
		hash := strings.TrimSpace(u.PasswordHash)
		if hash != u.PasswordHash {
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err == nil {
				goto ok
			}
		}
		pw := strings.TrimSpace(password)
		if pw != password {
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); err == nil {
				goto ok
			}
		}
		return nil, "", ErrInvalidCredentials
	}

ok:
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
