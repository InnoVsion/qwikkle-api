package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"qwikkle-api/internal/auth"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/db"
	"qwikkle-api/internal/server/docs"
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
	ID          string              `json:"id"`
	QKID        string              `json:"qkId"`
	Email       *string             `json:"email,omitempty"`
	Role        types.UserRole      `json:"role"`
	Status      types.AccountStatus `json:"status"`
	CreatedAt   time.Time           `json:"createdAt"`
	LastLoginAt *time.Time          `json:"lastLoginAt,omitempty"`
}

func New(cfg config.Config, pool *db.Pool, log *zap.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	origins := parseCSV(cfg.CORSAllowedOrigins)
	corsCfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	if len(origins) > 0 {
		corsCfg.AllowOrigins = origins
	} else {
		corsCfg.AllowOriginFunc = func(_ string) bool { return true }
	}
	r.Use(cors.New(corsCfg))

	docs.SwaggerInfo.Title = "Qwikkle API"
	docs.SwaggerInfo.Version = "0.1"
	docs.SwaggerInfo.BasePath = "/"

	repo := auth.NewPostgresRepository(pool)
	authService := auth.NewService(repo, cfg.JWTAccessSecret)

	if err := auth.BootstrapAdmin(context.Background(), repo); err != nil {
		log.Error("bootstrap admin failed", zap.Error(err))
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// @Summary Sign up
	// @Description Create a new user account
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param payload body signupRequest true "Signup payload"
	// @Success 201 {object} map[string]interface{}
	// @Failure 400 {object} map[string]string
	// @Failure 409 {object} map[string]string
	// @Failure 500 {object} map[string]string
	// @Router /signup [post]
	r.POST("/signup", func(c *gin.Context) {
		var req signupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, token, err := authService.Signup(c.Request.Context(), req.QKID, req.Email, req.Password)
		if err != nil {
			switch err {
			case auth.ErrIdentityTaken:
				c.JSON(http.StatusConflict, gin.H{"error": "qkId or email already in use"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
			}
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"user":  user,
			"token": token,
		})
	})

	// @Summary Login
	// @Description Log in with email and password
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param payload body loginRequest true "Login payload"
	// @Success 200 {object} map[string]interface{}
	// @Failure 400 {object} map[string]string
	// @Failure 401 {object} map[string]string
	// @Failure 500 {object} map[string]string
	// @Router /login [post]
	r.POST("/login", func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, token, err := authService.Login(c.Request.Context(), req.QKID, req.Password)
		if err != nil {
			switch err {
			case auth.ErrInvalidCredentials:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid qkId or password"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not log in"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user":  user,
			"token": token,
		})
	})

	adminAuth := r.Group("/admin/auth")
	adminAuth.POST("/login", func(c *gin.Context) {
		var req adminLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		normalizedQKID, err := types.NormalizeQKID(req.QKID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid qkId"})
			return
		}

		user, _, err := authService.Login(c.Request.Context(), normalizedQKID, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid qkId or password"})
			return
		}

		if user.Status != types.AccountStatusActive {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid qkId or password"})
			return
		}

		if user.Role != types.UserRoleAdmin && user.Role != types.UserRoleEditor {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid qkId or password"})
			return
		}

		accessToken, err := authService.GenerateAccessToken(user, 15*time.Minute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not log in"})
			return
		}

		refreshToken, refreshTokenHash, err := auth.NewRefreshToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not log in"})
			return
		}

		_, err = repo.CreateSession(
			c.Request.Context(),
			user.ID,
			refreshTokenHash,
			time.Now().Add(30*24*time.Hour),
			c.Request.UserAgent(),
			c.ClientIP(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not log in"})
			return
		}

		setAuthCookies(c, cfg, accessToken, refreshToken, 15*time.Minute, 30*24*time.Hour)
		c.JSON(http.StatusOK, gin.H{
			"admin": adminMeResponse{
				ID:          user.ID,
				QKID:        user.QKID,
				Email:       user.Email,
				Role:        user.Role,
				Status:      user.Status,
				CreatedAt:   user.CreatedAt,
				LastLoginAt: user.LastLoginAt,
			},
		})
	})

	adminAuth.POST("/refresh", func(c *gin.Context) {
		refreshToken, err := c.Cookie("refresh_token")
		if err != nil || refreshToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		session, err := repo.GetSessionByRefreshTokenHash(c.Request.Context(), auth.HashToken(refreshToken))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if session.RevokedAt != nil || time.Now().After(session.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, err := repo.GetUserByID(c.Request.Context(), session.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if user.Role != types.UserRoleAdmin && user.Role != types.UserRoleEditor {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		accessToken, err := authService.GenerateAccessToken(user, 15*time.Minute)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		newRefreshToken, newRefreshHash, err := auth.NewRefreshToken()
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if err := repo.RotateSession(c.Request.Context(), session.ID, newRefreshHash, time.Now().Add(30*24*time.Hour)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		setAuthCookies(c, cfg, accessToken, newRefreshToken, 15*time.Minute, 30*24*time.Hour)
		c.Status(http.StatusNoContent)
	})

	adminAuth.POST("/logout", func(c *gin.Context) {
		refreshToken, err := c.Cookie("refresh_token")
		if err == nil && refreshToken != "" {
			session, err := repo.GetSessionByRefreshTokenHash(c.Request.Context(), auth.HashToken(refreshToken))
			if err == nil {
				_ = repo.RevokeSession(c.Request.Context(), session.ID)
			}
		}

		clearAuthCookies(c, cfg)
		c.Status(http.StatusNoContent)
	})

	adminAuth.GET("/me", func(c *gin.Context) {
		accessToken, err := c.Cookie("access_token")
		if err != nil || accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := parseAccessToken(accessToken, cfg.JWTAccessSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		qkID, _ := claims["qkId"].(string)
		if qkID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, err := repo.GetUserByQKID(c.Request.Context(), qkID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if user.Role != types.UserRoleAdmin && user.Role != types.UserRoleEditor {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if user.Status != types.AccountStatusActive {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"admin": adminMeResponse{
				ID:          user.ID,
				QKID:        user.QKID,
				Email:       user.Email,
				Role:        user.Role,
				Status:      user.Status,
				CreatedAt:   user.CreatedAt,
				LastLoginAt: user.LastLoginAt,
			},
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
