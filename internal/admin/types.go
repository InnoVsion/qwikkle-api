package admin

import (
	"time"

	"qwikkle-api/internal/types"
)

type PaginationMeta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"totalPages"`
}

type PaginatedResponse[T any] struct {
	Data []T            `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

type AdminListUser struct {
	ID             string              `json:"id"`
	FirstName      string              `json:"firstName"`
	LastName       string              `json:"lastName"`
	Email          *string             `json:"email,omitempty"`
	Phone          *string             `json:"phone,omitempty"`
	AvatarURL      *string             `json:"avatarUrl,omitempty"`
	Status         types.AccountStatus `json:"status"`
	CreatedAt      time.Time           `json:"createdAt"`
	LastActiveAt   *time.Time          `json:"lastActiveAt,omitempty"`
	OrganizationID *string             `json:"organizationId,omitempty"`
}

type Organization struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	Email              *string                  `json:"email,omitempty"`
	Phone              *string                  `json:"phone,omitempty"`
	LogoURL            *string                  `json:"logoUrl,omitempty"`
	Status             types.AccountStatus      `json:"status"`
	VerificationStatus types.VerificationStatus `json:"verificationStatus"`
	MemberCount        int                      `json:"memberCount"`
	CreatedAt          time.Time                `json:"createdAt"`
	Documents          []OrganizationDocument   `json:"documents"`
}

type OrganizationDocument struct {
	ID                  string               `json:"id"`
	OrganizationID      string               `json:"organizationId"`
	OrganizationName    string               `json:"organizationName"`
	OrganizationLogoURL *string              `json:"organizationLogoUrl,omitempty"`
	Type                types.DocumentType   `json:"type"`
	StorageKey          string               `json:"storageKey"`
	FileName            string               `json:"fileName"`
	FileSize            int64                `json:"fileSize"`
	MimeType            string               `json:"mimeType"`
	DownloadURL         string               `json:"downloadUrl"`
	Status              types.DocumentStatus `json:"status"`
	RejectionReason     *string              `json:"rejectionReason,omitempty"`
	UploadedAt          time.Time            `json:"uploadedAt"`
	ReviewedAt          *time.Time           `json:"reviewedAt,omitempty"`
	ReviewedByID        *string              `json:"reviewedById,omitempty"`
}

type AccountActionPayload struct {
	Reason *string `json:"reason,omitempty"`
}

type DocumentRejectPayload struct {
	Reason string `json:"reason" binding:"required"`
}
