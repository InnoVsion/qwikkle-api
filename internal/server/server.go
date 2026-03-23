package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"qwikkle-api/internal/admin"
	"qwikkle-api/internal/auth"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/db"
	"qwikkle-api/internal/org"
	"qwikkle-api/internal/storage"
	"qwikkle-api/internal/types"
	"qwikkle-api/internal/uploads"
)

type Server struct {
	httpServer *http.Server
}

type signupRequest struct {
	QKID          string   `json:"qkId" binding:"required"`
	Email         *string  `json:"email" binding:"omitempty,email"`
	Password      string   `json:"password" binding:"required,min=6"`
	FirstName     *string  `json:"firstName"`
	LastName      *string  `json:"lastName"`
	Phone         *string  `json:"phone"`
	Gender        *string  `json:"gender"`
	DateOfBirth   *string  `json:"dateOfBirth"`
	Country       *string  `json:"country"`
	Interests     []string `json:"interests"`
	AvatarUploadID *string `json:"avatarUploadId"`
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
	adminRepo := admin.NewPostgresRepository(pool)

	uploadsRepo := uploads.NewPostgresRepository(pool)
	orgRepo := org.NewPostgresRepository(pool, uploadsRepo)

	presigner := storage.NewNoopPresigner()
	s3p, err := storage.NewS3Presigner(context.Background(), storage.S3Config{
		Region:          cfg.S3Region,
		Endpoint:        cfg.S3Endpoint,
		AccessKeyID:     cfg.S3AccessKeyID,
		SecretAccessKey: cfg.S3SecretAccessKey,
	})
	if err == nil {
		presigner = s3p
		log.Info("s3 presigner ready")
	} else {
		log.Warn("s3 presigner not configured", zap.Error(err))
	}

	if cfg.S3Bucket == "" {
		log.Warn("s3 bucket not configured; uploads disabled")
	} else {
		log.Info(
			"s3 uploads configured",
			zap.String("bucket", cfg.S3Bucket),
			zap.String("region", cfg.S3Region),
			zap.String("endpoint", cfg.S3Endpoint),
		)
	}

	r := NewRouter(cfg, repo, adminRepo, uploadsRepo, presigner, orgRepo, log)
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "down"})
			return
		}

		type tableCheck struct {
			Name   string `json:"name"`
			Exists bool   `json:"exists"`
		}

		checks := []tableCheck{
			{Name: "goose_db_version"},
			{Name: "users"},
			{Name: "sessions"},
			{Name: "organizations"},
			{Name: "organization_members"},
			{Name: "organization_documents"},
			{Name: "uploads"},
		}

		allTables := true
		for i := range checks {
			var exists bool
			if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+checks[i].Name).Scan(&exists); err != nil {
				allTables = false
				continue
			}
			checks[i].Exists = exists
			allTables = allTables && exists
		}

		status := "ok"
		code := http.StatusOK
		if !allTables {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, gin.H{
			"status": status,
			"db":     "ok",
			"tables": checks,
		})
	})

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
