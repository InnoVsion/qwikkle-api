package org

import (
	"context"

	"qwikkle-api/internal/types"
)

type SignupOrganizationInput struct {
	OwnerQKID     string
	OwnerPassword string

	OrganizationName         string
	OrganizationEmail        *string
	OrganizationPhone        *string
	OrganizationWebsite      *string
	OrganizationSocialHandle *string

	BusinessCertificateUploadID string
	RegistrantIDs               []RegistrantInput
}

type RegistrantInput struct {
	FullLegalName      string
	IDDocumentUploadID string
}

type Result struct {
	UserID             string
	OrganizationID     string
	VerificationStatus types.VerificationStatus
}

type Repository interface {
	SignupOrganization(ctx context.Context, in SignupOrganizationInput) (*Result, error)
}
