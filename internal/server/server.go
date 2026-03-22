package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"qwikkle-api/internal/auth"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/db"
	"qwikkle-api/internal/types"
)

type Server struct {
	httpServer *http.Server
}

type signupRequest struct {
	QKID     string  `json:"qkId" binding:"required"`
	Email    *string `json:"email" binding:"omitempty,email"`
	Password string  `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	QKID     string `json:"qkId" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type adminLoginRequest struct {
	QKID     string `json:"qkId" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type adminMeResponse struct {
	ID          string         `json:"id"`
	QKID        string         `json:"qkId"`
	Email       *string        `json:"email,omitempty"`
	Name        string         `json:"name"`
	Role        types.UserRole `json:"role"`
	CreatedAt   time.Time      `json:"createdAt"`
	LastLoginAt *time.Time     `json:"lastLoginAt,omitempty"`
}

func New(cfg config.Config, pool *db.Pool, log *zap.Logger) *Server {
	repo := auth.NewPostgresRepository(pool)
	r := NewRouter(cfg, repo, log)

	addr := fmt.Sprintf(":%s", cfg.Port)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	return &Server{
		httpServer: httpServer,
	}
}

func (s *Server) HTTPServer() *http.Server {
	return s.httpServer
}

func parseCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func setAuthCookies(c *gin.Context, cfg config.Config, accessToken string, refreshToken string, accessTTL time.Duration, refreshTTL time.Duration) {
	secure := cfg.AppEnv == "production"

	if secure {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}

	domain := cfg.CookieDomain
	accessMaxAge := int(accessTTL.Seconds())
	refreshMaxAge := int(refreshTTL.Seconds())

	c.SetCookie("access_token", accessToken, accessMaxAge, "/", domain, secure, true)
	c.SetCookie("refresh_token", refreshToken, refreshMaxAge, "/", domain, secure, true)
}

func clearAuthCookies(c *gin.Context, cfg config.Config) {
	secure := cfg.AppEnv == "production"
	if secure {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}

	domain := cfg.CookieDomain
	c.SetCookie("access_token", "", -1, "/", domain, secure, true)
	c.SetCookie("refresh_token", "", -1, "/", domain, secure, true)
}

func parseAccessToken(token string, secret string) (jwt.MapClaims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid token")
		}
		return []byte(secret), nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
