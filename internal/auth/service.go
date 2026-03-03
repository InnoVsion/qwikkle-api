package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}

type Service struct {
	mu         sync.RWMutex
	usersByEmail map[string]*User
	nextID     int64
	jwtSecret  string
}

func NewService(jwtSecret string) *Service {
	return &Service{
		usersByEmail: make(map[string]*User),
		nextID:     1,
		jwtSecret:  jwtSecret,
	}
}

var (
	ErrEmailTaken      = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

func (s *Service) Signup(email, password string) (*User, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.usersByEmail[email]; exists {
		return nil, "", ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	u := &User{
		ID:           s.nextID,
		Email:        email,
		PasswordHash: string(hash),
	}
	s.usersByEmail[email] = u
	s.nextID++

	token, err := s.generateToken(u)
	if err != nil {
		return nil, "", err
	}

	return u, token, nil
}

func (s *Service) Login(email, password string) (*User, string, error) {
	s.mu.RLock()
	u, ok := s.usersByEmail[email]
	s.mu.RUnlock()

	if !ok {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateToken(u)
	if err != nil {
		return nil, "", err
	}

	return u, token, nil
}

func (s *Service) generateToken(u *User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   u.ID,
		"email": u.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

