package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qwikkle-api/internal/config"
	"qwikkle-api/internal/org"
	"qwikkle-api/internal/storage"
	"qwikkle-api/internal/uploads"
)

type presignUploadRequest struct {
	FileName string `json:"fileName" binding:"required"`
	FileSize int64  `json:"fileSize" binding:"required"`
	MimeType string `json:"mimeType" binding:"required"`
}

type presignUploadResponse struct {
	UploadID   string                `json:"uploadId"`
	StorageKey string                `json:"storageKey"`
	Presign    storage.PresignResult `json:"presign"`
}

type completeUploadRequest struct {
	UploadID string `json:"uploadId" binding:"required"`
}

type signupOrganizationRequest struct {
	OwnerQKID     string  `json:"qkId" binding:"required"`
	OwnerPassword string  `json:"password" binding:"required,min=6"`
	OrgName       string  `json:"organizationName" binding:"required"`
	OrgEmail      *string `json:"organizationEmail" binding:"omitempty,email"`
	OrgPhone      *string `json:"organizationPhone"`
	OrgWebsite    *string `json:"organizationWebsite"`
	OrgSocial     *string `json:"organizationSocialHandle"`

	BusinessCertificateUploadID string `json:"businessCertificateUploadId" binding:"required"`
	Registrants                 []struct {
		FullLegalName      string `json:"fullLegalName" binding:"required"`
		IDDocumentUploadID string `json:"idDocumentUploadId" binding:"required"`
	} `json:"registrants" binding:"required"`
}

type signupOrganizationResponse struct {
	UserID         string `json:"userId"`
	OrganizationID string `json:"organizationId"`
}

func registerPhase3Routes(
	r *gin.Engine,
	cfg config.Config,
	uploadsRepo uploads.Repository,
	presigner storage.Presigner,
	orgRepo org.Repository,
) {
	r.POST("/uploads/presign", func(c *gin.Context) {
		var req presignUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if cfg.S3Bucket == "" {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "uploads not configured"})
			return
		}

		storageKey := buildStorageKey(req.FileName)
		upload, err := uploadsRepo.Create(c.Request.Context(), "s3", storageKey, nil, req.FileName, req.FileSize, req.MimeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create upload"})
			return
		}

		presign, err := presigner.PresignPutObject(
			c.Request.Context(),
			cfg.S3Bucket,
			storageKey,
			req.MimeType,
			req.FileSize,
			15*time.Minute,
		)
		if err != nil {
			if _, ok := err.(storage.NotConfiguredError); ok {
				c.JSON(http.StatusNotImplemented, gin.H{"error": "uploads not configured"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not presign upload"})
			return
		}

		c.JSON(http.StatusOK, presignUploadResponse{
			UploadID:   upload.ID,
			StorageKey: storageKey,
			Presign:    presign,
		})
	})

	r.POST("/uploads/complete", func(c *gin.Context) {
		var req completeUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := uploadsRepo.MarkCompleted(c.Request.Context(), req.UploadID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.POST("/signup/organization", func(c *gin.Context) {
		var req signupOrganizationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		registrants := make([]org.RegistrantInput, 0, len(req.Registrants))
		for _, r := range req.Registrants {
			registrants = append(registrants, org.RegistrantInput{
				FullLegalName:      r.FullLegalName,
				IDDocumentUploadID: r.IDDocumentUploadID,
			})
		}

		res, err := orgRepo.SignupOrganization(c.Request.Context(), org.SignupOrganizationInput{
			OwnerQKID:                   req.OwnerQKID,
			OwnerPassword:               req.OwnerPassword,
			OrganizationName:            req.OrgName,
			OrganizationEmail:           req.OrgEmail,
			OrganizationPhone:           req.OrgPhone,
			OrganizationWebsite:         req.OrgWebsite,
			OrganizationSocialHandle:    req.OrgSocial,
			BusinessCertificateUploadID: req.BusinessCertificateUploadID,
			RegistrantIDs:               registrants,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, signupOrganizationResponse{
			UserID:         res.UserID,
			OrganizationID: res.OrganizationID,
		})
	})
}

func buildStorageKey(fileName string) string {
	base := strings.TrimSpace(fileName)
	base = path.Base(base)
	if base == "." || base == "/" || base == "" {
		base = "file"
	}

	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)
	suffix := hex.EncodeToString(randBytes)

	return "uploads/" + time.Now().UTC().Format("20060102") + "/" + suffix + "/" + base
}
