package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qwikkle-api/internal/auth"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/types"
)

const adminUserContextKey = "adminUser"

func requireAdmin(cfg config.Config, repo auth.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := c.Cookie("access_token")
		if err != nil || accessToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := parseAccessToken(accessToken, cfg.JWTAccessSecret)
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
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if user.Status != types.AccountStatusActive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if user.Role != types.UserRoleAdmin && user.Role != types.UserRoleEditor {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set(adminUserContextKey, user)
		c.Next()
	}
}

func getAdminUser(c *gin.Context) (*auth.User, bool) {
	v, ok := c.Get(adminUserContextKey)
	if !ok {
		return nil, false
	}
	u, ok := v.(*auth.User)
	return u, ok
}
