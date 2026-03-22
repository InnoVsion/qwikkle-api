package admin

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"qwikkle-api/internal/db"
	"qwikkle-api/internal/types"
)

var ErrNotFound = errors.New("not found")

type PostgresRepository struct {
	pool *db.Pool
}

func NewPostgresRepository(pool *db.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func clampPagination(page int, limit int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit
	return page, limit, offset
}

func (r *PostgresRepository) ListUsers(ctx context.Context, params ListUsersParams) (PaginatedResponse[AdminListUser], error) {
	page, limit, offset := clampPagination(params.Page, params.Limit)

	var where []string
	args := make([]any, 0, 6)

	where = append(where, "role = 'user'")

	if params.Search != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(params.Search))+"%")
		where = append(where, "(LOWER(qk_id::text) LIKE $"+itoa(len(args))+" OR LOWER(COALESCE(email::text, '')) LIKE $"+itoa(len(args))+" OR LOWER(first_name) LIKE $"+itoa(len(args))+" OR LOWER(last_name) LIKE $"+itoa(len(args))+")")
	}

	if params.Status != "" {
		args = append(args, string(params.Status))
		where = append(where, "status = $"+itoa(len(args))+"::account_status")
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	countQ := "SELECT COUNT(*) FROM users " + whereSQL
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return PaginatedResponse[AdminListUser]{}, err
	}

	q := `
		SELECT id::text, first_name, last_name, email::text, phone, avatar_url, status::text, created_at, last_active_at, organization_id::text
		FROM users
	` + whereSQL + `
		ORDER BY created_at DESC
		LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return PaginatedResponse[AdminListUser]{}, err
	}
	defer rows.Close()

	users := make([]AdminListUser, 0)
	for rows.Next() {
		var u AdminListUser
		var email pgtype.Text
		var orgID pgtype.Text
		if err := rows.Scan(
			&u.ID,
			&u.FirstName,
			&u.LastName,
			&email,
			&u.Phone,
			&u.AvatarURL,
			&u.Status,
			&u.CreatedAt,
			&u.LastActiveAt,
			&orgID,
		); err != nil {
			return PaginatedResponse[AdminListUser]{}, err
		}
		if email.Valid {
			u.Email = &email.String
		}
		if orgID.Valid {
			u.OrganizationID = &orgID.String
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return PaginatedResponse[AdminListUser]{}, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return PaginatedResponse[AdminListUser]{
		Data: users,
		Meta: PaginationMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages},
	}, nil
}

func (r *PostgresRepository) GetUser(ctx context.Context, id string) (*AdminListUser, error) {
	const q = `
		SELECT id::text, first_name, last_name, email::text, phone, avatar_url, status::text, created_at, last_active_at, organization_id::text
		FROM users
		WHERE id = $1
	`
	var u AdminListUser
	var email pgtype.Text
	var orgID pgtype.Text
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&email,
		&u.Phone,
		&u.AvatarURL,
		&u.Status,
		&u.CreatedAt,
		&u.LastActiveAt,
		&orgID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if email.Valid {
		u.Email = &email.String
	}
	if orgID.Valid {
		u.OrganizationID = &orgID.String
	}
	return &u, nil
}

func (r *PostgresRepository) UpdateUserStatus(ctx context.Context, id string, status types.AccountStatus) error {
	const q = `UPDATE users SET status = $1::account_status WHERE id = $2`
	ct, err := r.pool.Exec(ctx, q, string(status), id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteUser(ctx context.Context, id string) error {
	const q = `DELETE FROM users WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListOrganizations(ctx context.Context, params ListOrganizationsParams) (PaginatedResponse[Organization], error) {
	page, limit, offset := clampPagination(params.Page, params.Limit)

	var where []string
	args := make([]any, 0, 6)

	if params.Search != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(params.Search))+"%")
		where = append(where, "(LOWER(name) LIKE $"+itoa(len(args))+" OR LOWER(COALESCE(email::text, '')) LIKE $"+itoa(len(args))+")")
	}
	if params.Status != "" {
		args = append(args, string(params.Status))
		where = append(where, "status = $"+itoa(len(args))+"::account_status")
	}
	if params.VerificationStatus != "" {
		args = append(args, string(params.VerificationStatus))
		where = append(where, "verification_status = $"+itoa(len(args))+"::verification_status")
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	countQ := "SELECT COUNT(*) FROM organizations " + whereSQL
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return PaginatedResponse[Organization]{}, err
	}

	q := `
		SELECT
			o.id::text,
			o.name,
			o.email::text,
			o.phone,
			o.logo_url,
			o.status::text,
			o.verification_status::text,
			COALESCE((SELECT COUNT(*) FROM organization_members m WHERE m.organization_id = o.id), 0) AS member_count,
			o.created_at,
			COALESCE(
				json_agg(
					json_build_object(
						'id', d.id::text,
						'organizationId', d.organization_id::text,
						'organizationName', o.name,
						'organizationLogoUrl', o.logo_url,
						'type', d.type::text,
						'storageKey', d.storage_key,
						'fileName', d.file_name,
						'fileSize', d.file_size,
						'mimeType', d.mime_type,
						'downloadUrl', COALESCE(d.download_url, d.storage_key),
						'status', d.status::text,
						'rejectionReason', d.rejection_reason,
						'uploadedAt', d.uploaded_at,
						'reviewedAt', d.reviewed_at,
						'reviewedById', d.reviewed_by_user_id::text
					)
				) FILTER (WHERE d.id IS NOT NULL),
				'[]'::json
			) AS documents
		FROM organizations o
		LEFT JOIN organization_documents d ON d.organization_id = o.id
	` + whereSQL + `
		GROUP BY o.id
		ORDER BY o.created_at DESC
		LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return PaginatedResponse[Organization]{}, err
	}
	defer rows.Close()

	orgs := make([]Organization, 0)
	for rows.Next() {
		var o Organization
		var email pgtype.Text
		var docsBytes []byte
		if err := rows.Scan(
			&o.ID,
			&o.Name,
			&email,
			&o.Phone,
			&o.LogoURL,
			&o.Status,
			&o.VerificationStatus,
			&o.MemberCount,
			&o.CreatedAt,
			&docsBytes,
		); err != nil {
			return PaginatedResponse[Organization]{}, err
		}
		if email.Valid {
			o.Email = &email.String
		}
		if err := json.Unmarshal(docsBytes, &o.Documents); err != nil {
			return PaginatedResponse[Organization]{}, err
		}
		orgs = append(orgs, o)
	}
	if err := rows.Err(); err != nil {
		return PaginatedResponse[Organization]{}, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return PaginatedResponse[Organization]{
		Data: orgs,
		Meta: PaginationMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages},
	}, nil
}

func (r *PostgresRepository) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	const q = `
		SELECT
			o.id::text,
			o.name,
			o.email::text,
			o.phone,
			o.logo_url,
			o.status::text,
			o.verification_status::text,
			COALESCE((SELECT COUNT(*) FROM organization_members m WHERE m.organization_id = o.id), 0) AS member_count,
			o.created_at,
			COALESCE(
				json_agg(
					json_build_object(
						'id', d.id::text,
						'organizationId', d.organization_id::text,
						'organizationName', o.name,
						'organizationLogoUrl', o.logo_url,
						'type', d.type::text,
						'storageKey', d.storage_key,
						'fileName', d.file_name,
						'fileSize', d.file_size,
						'mimeType', d.mime_type,
						'downloadUrl', COALESCE(d.download_url, d.storage_key),
						'status', d.status::text,
						'rejectionReason', d.rejection_reason,
						'uploadedAt', d.uploaded_at,
						'reviewedAt', d.reviewed_at,
						'reviewedById', d.reviewed_by_user_id::text
					)
				) FILTER (WHERE d.id IS NOT NULL),
				'[]'::json
			) AS documents
		FROM organizations o
		LEFT JOIN organization_documents d ON d.organization_id = o.id
		WHERE o.id = $1
		GROUP BY o.id
	`

	var o Organization
	var email pgtype.Text
	var docsBytes []byte
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&o.ID,
		&o.Name,
		&email,
		&o.Phone,
		&o.LogoURL,
		&o.Status,
		&o.VerificationStatus,
		&o.MemberCount,
		&o.CreatedAt,
		&docsBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if email.Valid {
		o.Email = &email.String
	}
	if err := json.Unmarshal(docsBytes, &o.Documents); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *PostgresRepository) UpdateOrganizationStatus(ctx context.Context, id string, status types.AccountStatus) error {
	const q = `UPDATE organizations SET status = $1::account_status WHERE id = $2`
	ct, err := r.pool.Exec(ctx, q, string(status), id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteOrganization(ctx context.Context, id string) error {
	const q = `DELETE FROM organizations WHERE id = $1`
	ct, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListDocuments(ctx context.Context, params ListDocumentsParams) (PaginatedResponse[OrganizationDocument], error) {
	page, limit, offset := clampPagination(params.Page, params.Limit)

	var where []string
	args := make([]any, 0, 6)

	if params.Search != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(params.Search))+"%")
		where = append(where, "(LOWER(d.file_name) LIKE $"+itoa(len(args))+" OR LOWER(o.name) LIKE $"+itoa(len(args))+")")
	}
	if params.Status != "" {
		args = append(args, string(params.Status))
		where = append(where, "d.status = $"+itoa(len(args))+"::document_status")
	}
	if params.Type != "" {
		args = append(args, string(params.Type))
		where = append(where, "d.type = $"+itoa(len(args))+"::document_type")
	}
	if params.OrgID != "" {
		args = append(args, params.OrgID)
		where = append(where, "d.organization_id = $"+itoa(len(args))+"::uuid")
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	countQ := `
		SELECT COUNT(*)
		FROM organization_documents d
		JOIN organizations o ON o.id = d.organization_id
	` + whereSQL
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return PaginatedResponse[OrganizationDocument]{}, err
	}

	q := `
		SELECT
			d.id::text,
			d.organization_id::text,
			o.name,
			o.logo_url,
			d.type::text,
			d.storage_key,
			d.file_name,
			d.file_size,
			d.mime_type,
			COALESCE(d.download_url, d.storage_key),
			d.status::text,
			d.rejection_reason,
			d.uploaded_at,
			d.reviewed_at,
			d.reviewed_by_user_id::text
		FROM organization_documents d
		JOIN organizations o ON o.id = d.organization_id
	` + whereSQL + `
		ORDER BY d.uploaded_at DESC
		LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return PaginatedResponse[OrganizationDocument]{}, err
	}
	defer rows.Close()

	docs := make([]OrganizationDocument, 0)
	for rows.Next() {
		var d OrganizationDocument
		var reviewedBy pgtype.Text
		if err := rows.Scan(
			&d.ID,
			&d.OrganizationID,
			&d.OrganizationName,
			&d.OrganizationLogoURL,
			&d.Type,
			&d.StorageKey,
			&d.FileName,
			&d.FileSize,
			&d.MimeType,
			&d.DownloadURL,
			&d.Status,
			&d.RejectionReason,
			&d.UploadedAt,
			&d.ReviewedAt,
			&reviewedBy,
		); err != nil {
			return PaginatedResponse[OrganizationDocument]{}, err
		}
		if reviewedBy.Valid {
			d.ReviewedByID = &reviewedBy.String
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return PaginatedResponse[OrganizationDocument]{}, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return PaginatedResponse[OrganizationDocument]{
		Data: docs,
		Meta: PaginationMeta{Total: total, Page: page, Limit: limit, TotalPages: totalPages},
	}, nil
}

func (r *PostgresRepository) GetDocument(ctx context.Context, id string) (*OrganizationDocument, error) {
	const q = `
		SELECT
			d.id::text,
			d.organization_id::text,
			o.name,
			o.logo_url,
			d.type::text,
			d.storage_key,
			d.file_name,
			d.file_size,
			d.mime_type,
			COALESCE(d.download_url, d.storage_key),
			d.status::text,
			d.rejection_reason,
			d.uploaded_at,
			d.reviewed_at,
			d.reviewed_by_user_id::text
		FROM organization_documents d
		JOIN organizations o ON o.id = d.organization_id
		WHERE d.id = $1
	`

	var d OrganizationDocument
	var reviewedBy pgtype.Text
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&d.ID,
		&d.OrganizationID,
		&d.OrganizationName,
		&d.OrganizationLogoURL,
		&d.Type,
		&d.StorageKey,
		&d.FileName,
		&d.FileSize,
		&d.MimeType,
		&d.DownloadURL,
		&d.Status,
		&d.RejectionReason,
		&d.UploadedAt,
		&d.ReviewedAt,
		&reviewedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if reviewedBy.Valid {
		d.ReviewedByID = &reviewedBy.String
	}
	return &d, nil
}

func (r *PostgresRepository) ApproveDocument(ctx context.Context, id string, reviewerID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orgID string
	err = tx.QueryRow(ctx, `SELECT organization_id::text FROM organization_documents WHERE id = $1`, id).Scan(&orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE organization_documents
		SET status = 'approved'::document_status,
		    reviewed_at = NOW(),
		    reviewed_by_user_id = $2
		WHERE id = $1
	`, id, reviewerID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE organizations
		SET verification_status = 'approved'::verification_status
		WHERE id = $1
	`, orgID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) RejectDocument(ctx context.Context, id string, reviewerID string, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orgID string
	err = tx.QueryRow(ctx, `SELECT organization_id::text FROM organization_documents WHERE id = $1`, id).Scan(&orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE organization_documents
		SET status = 'rejected'::document_status,
		    rejection_reason = $3,
		    reviewed_at = NOW(),
		    reviewed_by_user_id = $2
		WHERE id = $1
	`, id, reviewerID, reason)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE organizations
		SET verification_status = 'rejected'::verification_status
		WHERE id = $1
	`, orgID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
