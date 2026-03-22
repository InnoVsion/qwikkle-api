package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qwikkle-api/internal/admin"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/storage"
	"qwikkle-api/internal/types"
)

func registerAdminRoutes(r *gin.RouterGroup, repo admin.Repository, cfg config.Config, presigner storage.Presigner) {
	r.GET("/users", func(c *gin.Context) {
		params, ok := parseListUsersParams(c)
		if !ok {
			return
		}
		res, err := repo.ListUsers(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, res)
	})

	r.GET("/users/:id", func(c *gin.Context) {
		u, err := repo.GetUser(c.Request.Context(), c.Param("id"))
		if err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, u)
	})

	r.PATCH("/users/:id/suspend", func(c *gin.Context) {
		var payload admin.AccountActionPayload
		_ = c.ShouldBindJSON(&payload)
		_ = payload

		if err := repo.UpdateUserStatus(c.Request.Context(), c.Param("id"), types.AccountStatusSuspended); err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.PATCH("/users/:id/deactivate", func(c *gin.Context) {
		var payload admin.AccountActionPayload
		_ = c.ShouldBindJSON(&payload)
		_ = payload

		if err := repo.UpdateUserStatus(c.Request.Context(), c.Param("id"), types.AccountStatusDeactivated); err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.DELETE("/users/:id", func(c *gin.Context) {
		if err := repo.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.GET("/organizations", func(c *gin.Context) {
		params, ok := parseListOrganizationsParams(c)
		if !ok {
			return
		}
		res, err := repo.ListOrganizations(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, res)
	})

	r.GET("/organizations/:id", func(c *gin.Context) {
		o, err := repo.GetOrganization(c.Request.Context(), c.Param("id"))
		if err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, o)
	})

	r.PATCH("/organizations/:id/suspend", func(c *gin.Context) {
		var payload admin.AccountActionPayload
		_ = c.ShouldBindJSON(&payload)
		_ = payload

		if err := repo.UpdateOrganizationStatus(c.Request.Context(), c.Param("id"), types.AccountStatusSuspended); err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.PATCH("/organizations/:id/deactivate", func(c *gin.Context) {
		var payload admin.AccountActionPayload
		_ = c.ShouldBindJSON(&payload)
		_ = payload

		if err := repo.UpdateOrganizationStatus(c.Request.Context(), c.Param("id"), types.AccountStatusDeactivated); err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.DELETE("/organizations/:id", func(c *gin.Context) {
		if err := repo.DeleteOrganization(c.Request.Context(), c.Param("id")); err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.GET("/documents", func(c *gin.Context) {
		params, ok := parseListDocumentsParams(c)
		if !ok {
			return
		}
		res, err := repo.ListDocuments(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, res)
	})

	r.GET("/documents/:id", func(c *gin.Context) {
		d, err := repo.GetDocument(c.Request.Context(), c.Param("id"))
		if err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.JSON(http.StatusOK, d)
	})

	r.GET("/documents/:id/download", func(c *gin.Context) {
		if cfg.S3Bucket == "" {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "downloads not configured"})
			return
		}

		d, err := repo.GetDocument(c.Request.Context(), c.Param("id"))
		if err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if d.StorageKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "document has no storage key"})
			return
		}

		p, err := presigner.PresignGetObject(c.Request.Context(), cfg.S3Bucket, d.StorageKey, 10*time.Minute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create download url"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"url":     p.URL,
			"expires": p.Expires,
		})
	})

	r.PATCH("/documents/:id/approve", func(c *gin.Context) {
		adminUser, ok := getAdminUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if err := repo.ApproveDocument(c.Request.Context(), c.Param("id"), adminUser.ID); err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.PATCH("/documents/:id/reject", func(c *gin.Context) {
		adminUser, ok := getAdminUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var payload admin.DocumentRejectPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := repo.RejectDocument(c.Request.Context(), c.Param("id"), adminUser.ID, payload.Reason); err != nil {
			if err == admin.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		c.Status(http.StatusNoContent)
	})
}

func parseListUsersParams(c *gin.Context) (admin.ListUsersParams, bool) {
	page, limit := parsePageLimit(c)

	var status types.AccountStatus
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		switch v {
		case string(types.AccountStatusActive), string(types.AccountStatusSuspended), string(types.AccountStatusDeactivated):
			status = types.AccountStatus(v)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return admin.ListUsersParams{}, false
		}
	}

	return admin.ListUsersParams{
		Search:    c.Query("search"),
		Status:    status,
		DateRange: c.Query("dateRange"),
		Page:      page,
		Limit:     limit,
	}, true
}

func parseListOrganizationsParams(c *gin.Context) (admin.ListOrganizationsParams, bool) {
	page, limit := parsePageLimit(c)

	var status types.AccountStatus
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		switch v {
		case string(types.AccountStatusActive), string(types.AccountStatusSuspended), string(types.AccountStatusDeactivated):
			status = types.AccountStatus(v)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return admin.ListOrganizationsParams{}, false
		}
	}

	var verificationStatus types.VerificationStatus
	if v := strings.TrimSpace(c.Query("verificationStatus")); v != "" {
		switch v {
		case string(types.VerificationStatusPending), string(types.VerificationStatusApproved), string(types.VerificationStatusRejected):
			verificationStatus = types.VerificationStatus(v)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verificationStatus"})
			return admin.ListOrganizationsParams{}, false
		}
	}

	return admin.ListOrganizationsParams{
		Search:             c.Query("search"),
		Status:             status,
		VerificationStatus: verificationStatus,
		Page:               page,
		Limit:              limit,
	}, true
}

func parseListDocumentsParams(c *gin.Context) (admin.ListDocumentsParams, bool) {
	page, limit := parsePageLimit(c)

	var status types.DocumentStatus
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		switch v {
		case string(types.DocumentStatusPending), string(types.DocumentStatusApproved), string(types.DocumentStatusRejected):
			status = types.DocumentStatus(v)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return admin.ListDocumentsParams{}, false
		}
	}

	var docType types.DocumentType
	if v := strings.TrimSpace(c.Query("type")); v != "" {
		switch v {
		case string(types.DocumentTypeRegistrationCertificate),
			string(types.DocumentTypeTaxID),
			string(types.DocumentTypeProofOfAddress),
			string(types.DocumentTypeIDDocument),
			string(types.DocumentTypeOther):
			docType = types.DocumentType(v)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
			return admin.ListDocumentsParams{}, false
		}
	}

	return admin.ListDocumentsParams{
		Search: c.Query("search"),
		Status: status,
		Type:   docType,
		OrgID:  c.Query("orgId"),
		Page:   page,
		Limit:  limit,
	}, true
}

func parsePageLimit(c *gin.Context) (int, int) {
	page := 1
	limit := 20

	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			page = n
		}
	}
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	return page, limit
}
