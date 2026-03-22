package admin

import (
	"context"

	"qwikkle-api/internal/types"
)

type ListUsersParams struct {
	Search    string
	Status    types.AccountStatus
	Page      int
	Limit     int
	DateRange string
}

type ListOrganizationsParams struct {
	Search             string
	Status             types.AccountStatus
	VerificationStatus types.VerificationStatus
	Page               int
	Limit              int
}

type ListDocumentsParams struct {
	Search string
	Status types.DocumentStatus
	Type   types.DocumentType
	OrgID  string
	Page   int
	Limit  int
}

type Repository interface {
	ListUsers(ctx context.Context, params ListUsersParams) (PaginatedResponse[AdminListUser], error)
	GetUser(ctx context.Context, id string) (*AdminListUser, error)
	UpdateUserStatus(ctx context.Context, id string, status types.AccountStatus) error
	DeleteUser(ctx context.Context, id string) error

	ListOrganizations(ctx context.Context, params ListOrganizationsParams) (PaginatedResponse[Organization], error)
	GetOrganization(ctx context.Context, id string) (*Organization, error)
	UpdateOrganizationStatus(ctx context.Context, id string, status types.AccountStatus) error
	DeleteOrganization(ctx context.Context, id string) error

	ListDocuments(ctx context.Context, params ListDocumentsParams) (PaginatedResponse[OrganizationDocument], error)
	GetDocument(ctx context.Context, id string) (*OrganizationDocument, error)
	ApproveDocument(ctx context.Context, id string, reviewerID string) error
	RejectDocument(ctx context.Context, id string, reviewerID string, reason string) error
}
