package org

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"qwikkle-api/internal/db"
	"qwikkle-api/internal/types"
	"qwikkle-api/internal/uploads"
)

var ErrInvalidUpload = errors.New("invalid upload")

type PostgresRepository struct {
	pool        *db.Pool
	uploadsRepo uploads.Repository
}

func NewPostgresRepository(pool *db.Pool, uploadsRepo uploads.Repository) *PostgresRepository {
	return &PostgresRepository{pool: pool, uploadsRepo: uploadsRepo}
}

func (r *PostgresRepository) SignupOrganization(ctx context.Context, in SignupOrganizationInput) (*Result, error) {
	ownerQKID, err := types.NormalizeQKID(in.OwnerQKID)
	if err != nil {
		return nil, err
	}
	if in.OwnerPassword == "" {
		return nil, errors.New("password is required")
	}
	if in.OrganizationName == "" {
		return nil, errors.New("organization name is required")
	}
	if in.BusinessCertificateUploadID == "" {
		return nil, errors.New("businessCertificateUploadId is required")
	}

	businessUpload, err := r.uploadsRepo.Get(ctx, in.BusinessCertificateUploadID)
	if err != nil || businessUpload.Status != uploads.UploadStatusCompleted {
		return nil, ErrInvalidUpload
	}

	for _, reg := range in.RegistrantIDs {
		if reg.FullLegalName == "" || reg.IDDocumentUploadID == "" {
			return nil, errors.New("invalid registrant")
		}
		u, err := r.uploadsRepo.Get(ctx, reg.IDDocumentUploadID)
		if err != nil || u.Status != uploads.UploadStatusCompleted {
			return nil, ErrInvalidUpload
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.OwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (qk_id, password_hash, role, status)
		VALUES ($1, $2, 'user'::user_role, 'active'::account_status)
		RETURNING id::text
	`, ownerQKID, string(hash)).Scan(&userID)
	if err != nil {
		return nil, err
	}

	var orgID string
	err = tx.QueryRow(ctx, `
		INSERT INTO organizations (owner_user_id, name, email, phone, website, social_handle, status, verification_status)
		VALUES ($1, $2, $3, $4, $5, $6, 'active'::account_status, 'pending'::verification_status)
		RETURNING id::text
	`, userID, in.OrganizationName, in.OrganizationEmail, in.OrganizationPhone, in.OrganizationWebsite, in.OrganizationSocialHandle).Scan(&orgID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, userID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT DO NOTHING
	`, orgID, userID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO organization_documents (
			organization_id,
			uploaded_by_user_id,
			type,
			status,
			file_name,
			file_size,
			mime_type,
			storage_key,
			download_url,
			uploaded_at
		)
		VALUES ($1, $2, $3::document_type, 'pending'::document_status, $4, $5, $6, $7, $8, NOW())
	`, orgID, userID, string(types.DocumentTypeRegistrationCertificate), businessUpload.FileName, businessUpload.FileSize, businessUpload.MimeType, businessUpload.StorageKey, businessUpload.StorageKey)
	if err != nil {
		return nil, err
	}

	for _, reg := range in.RegistrantIDs {
		u, _ := r.uploadsRepo.Get(ctx, reg.IDDocumentUploadID)
		_, err = tx.Exec(ctx, `
			INSERT INTO organization_documents (
				organization_id,
				uploaded_by_user_id,
				type,
				status,
				file_name,
				file_size,
				mime_type,
				storage_key,
				download_url,
				uploaded_at
			)
			VALUES ($1, $2, $3::document_type, 'pending'::document_status, $4, $5, $6, $7, $8, NOW())
		`, orgID, userID, string(types.DocumentTypeIDDocument), u.FileName, u.FileSize, u.MimeType, u.StorageKey, u.StorageKey)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE uploads SET status = 'completed'::upload_status, completed_at = COALESCE(completed_at, NOW()) WHERE id = $1`, reg.IDDocumentUploadID)
		if err != nil {
			return nil, err
		}
	}

	_, err = tx.Exec(ctx, `UPDATE uploads SET status = 'completed'::upload_status, completed_at = COALESCE(completed_at, NOW()) WHERE id = $1`, in.BusinessCertificateUploadID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &Result{
		UserID:             userID,
		OrganizationID:     orgID,
		VerificationStatus: types.VerificationStatusPending,
	}, nil
}
