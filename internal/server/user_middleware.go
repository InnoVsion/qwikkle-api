package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"qwikkle-api/internal/auth"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/types"
)

const userContextKey = "user"

func requireUserToken(cfg config.Config, repo auth.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		token := strings.TrimSpace(h[7:])
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := parseAccessToken(token, cfg.JWTAccessSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		qkID, _ := claims["qkId"].(string)
		if qkID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, err := repo.GetUserByQKID(c.Request.Context(), qkID)
		if err != nil || user.Status != types.AccountStatusActive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set(userContextKey, user)
		c.Next()
	}
}

func getUser(c *gin.Context) (*auth.User, bool) {
	v, ok := c.Get(userContextKey)
	if !ok {
		return nil, false
	}
	u, ok := v.(*auth.User)
	return u, ok
}

