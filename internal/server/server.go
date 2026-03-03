package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	"qwikkle-api/internal/auth"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/server/docs"
)

type Server struct {
	httpServer *http.Server
}

type signupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func New(cfg config.Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	docs.SwaggerInfo.Title = "Qwikkle API"
	docs.SwaggerInfo.Version = "0.1"
	docs.SwaggerInfo.BasePath = "/"

	authService := auth.NewService(cfg.JWTAccessSecret)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(files.Handler))

	r.POST("/signup", func(c *gin.Context) {
		var req signupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, token, err := authService.Signup(req.Email, req.Password)
		if err != nil {
			switch err {
			case auth.ErrEmailTaken:
				c.JSON(http.StatusConflict, gin.H{"error": "email already in use"})
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

	r.POST("/login", func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, token, err := authService.Login(req.Email, req.Password)
		if err != nil {
			switch err {
			case auth.ErrInvalidCredentials:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
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

