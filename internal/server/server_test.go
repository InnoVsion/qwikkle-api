package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"qwikkle-api/internal/admin"
	"qwikkle-api/internal/auth"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/org"
	"qwikkle-api/internal/storage"
	"qwikkle-api/internal/types"
	"qwikkle-api/internal/uploads"
)

type memoryAuthRepo struct {
	mu sync.Mutex

	nextUserID    int
	nextSessionID int

	usersByID    map[string]*auth.User
	usersByQKID  map[string]*auth.User
	emailToUser  map[string]*auth.User
	sessionsByID map[string]*auth.Session
	sessionsByHT map[string]*auth.Session
}

func newMemoryAuthRepo() *memoryAuthRepo {
	return &memoryAuthRepo{
		usersByID:    map[string]*auth.User{},
		usersByQKID:  map[string]*auth.User{},
		emailToUser:  map[string]*auth.User{},
		sessionsByID: map[string]*auth.Session{},
		sessionsByHT: map[string]*auth.Session{},
	}
}

func (r *memoryAuthRepo) CreateUser(
	ctx context.Context,
	qkID string,
	email *string,
	passwordHash string,
	role string,
) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.usersByQKID[qkID]; ok {
		return nil, auth.ErrIdentityTaken
	}
	if email != nil && *email != "" {
		if _, ok := r.emailToUser[*email]; ok {
			return nil, auth.ErrIdentityTaken
		}
	}

	r.nextUserID++
	id := fmt.Sprintf("u_%d", r.nextUserID)

	now := time.Now()
	u := &auth.User{
		ID:           id,
		QKID:         qkID,
		Email:        email,
		Role:         types.UserRole(role),
		Status:       types.AccountStatusActive,
		CreatedAt:    now,
		LastLoginAt:  nil,
		PasswordHash: passwordHash,
	}

	r.usersByID[id] = u
	r.usersByQKID[qkID] = u
	if email != nil && *email != "" {
		r.emailToUser[*email] = u
	}

	return u, nil
}

func (r *memoryAuthRepo) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.usersByID[id]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return u, nil
}

func (r *memoryAuthRepo) GetUserByQKID(ctx context.Context, qkID string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.usersByQKID[qkID]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return u, nil
}

func (r *memoryAuthRepo) CreateSession(
	ctx context.Context,
	userID string,
	refreshTokenHash string,
	expiresAt time.Time,
	userAgent string,
	ip string,
) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextSessionID++
	id := fmt.Sprintf("s_%d", r.nextSessionID)
	now := time.Now()

	var uaPtr *string
	if userAgent != "" {
		v := userAgent
		uaPtr = &v
	}
	var ipPtr *string
	if ip != "" {
		v := ip
		ipPtr = &v
	}

	s := &auth.Session{
		ID:               id,
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		CreatedAt:        now,
		ExpiresAt:        expiresAt,
		RevokedAt:        nil,
		UserAgent:        uaPtr,
		IP:               ipPtr,
	}

	r.sessionsByID[id] = s
	r.sessionsByHT[refreshTokenHash] = s

	return id, nil
}

func (r *memoryAuthRepo) GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*auth.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessionsByHT[refreshTokenHash]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}
	return s, nil
}

func (r *memoryAuthRepo) RotateSession(ctx context.Context, sessionID string, refreshTokenHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessionsByID[sessionID]
	if !ok || s.RevokedAt != nil {
		return auth.ErrSessionNotFound
	}

	delete(r.sessionsByHT, s.RefreshTokenHash)
	s.RefreshTokenHash = refreshTokenHash
	s.ExpiresAt = expiresAt
	r.sessionsByHT[refreshTokenHash] = s

	return nil
}

func (r *memoryAuthRepo) RevokeSession(ctx context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessionsByID[sessionID]
	if !ok || s.RevokedAt != nil {
		return auth.ErrSessionNotFound
	}

	now := time.Now()
	s.RevokedAt = &now
	return nil
}

type memoryAdminRepo struct {
	auth *memoryAuthRepo
	mu   sync.Mutex

	orgs map[string]*admin.Organization
	docs map[string]*admin.OrganizationDocument
}

func newMemoryAdminRepo(authRepo *memoryAuthRepo) *memoryAdminRepo {
	return &memoryAdminRepo{
		auth: authRepo,
		orgs: map[string]*admin.Organization{},
		docs: map[string]*admin.OrganizationDocument{},
	}
}

func (r *memoryAdminRepo) ListUsers(ctx context.Context, params admin.ListUsersParams) (admin.PaginatedResponse[admin.AdminListUser], error) {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()

	out := make([]admin.AdminListUser, 0)
	for _, u := range r.auth.usersByID {
		if u.Role != types.UserRoleUser {
			continue
		}
		out = append(out, admin.AdminListUser{
			ID:             u.ID,
			FirstName:      "",
			LastName:       "",
			Email:          u.Email,
			Status:         u.Status,
			CreatedAt:      u.CreatedAt,
			LastActiveAt:   u.LastLoginAt,
			OrganizationID: nil,
		})
	}

	return admin.PaginatedResponse[admin.AdminListUser]{
		Data: out,
		Meta: admin.PaginationMeta{
			Total:      len(out),
			Page:       1,
			Limit:      len(out),
			TotalPages: 1,
		},
	}, nil
}

func (r *memoryAdminRepo) GetUser(ctx context.Context, id string) (*admin.AdminListUser, error) {
	u, err := r.auth.GetUserByID(ctx, id)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return nil, admin.ErrNotFound
		}
		return nil, err
	}
	return &admin.AdminListUser{
		ID:             u.ID,
		FirstName:      "",
		LastName:       "",
		Email:          u.Email,
		Status:         u.Status,
		CreatedAt:      u.CreatedAt,
		LastActiveAt:   u.LastLoginAt,
		OrganizationID: nil,
	}, nil
}

func (r *memoryAdminRepo) UpdateUserStatus(ctx context.Context, id string, status types.AccountStatus) error {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()

	u, ok := r.auth.usersByID[id]
	if !ok {
		return admin.ErrNotFound
	}
	u.Status = status
	return nil
}

func (r *memoryAdminRepo) DeleteUser(ctx context.Context, id string) error {
	r.auth.mu.Lock()
	defer r.auth.mu.Unlock()

	u, ok := r.auth.usersByID[id]
	if !ok {
		return admin.ErrNotFound
	}
	delete(r.auth.usersByID, id)
	delete(r.auth.usersByQKID, u.QKID)
	if u.Email != nil && *u.Email != "" {
		delete(r.auth.emailToUser, *u.Email)
	}
	return nil
}

func (r *memoryAdminRepo) ListOrganizations(ctx context.Context, params admin.ListOrganizationsParams) (admin.PaginatedResponse[admin.Organization], error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]admin.Organization, 0, len(r.orgs))
	for _, o := range r.orgs {
		copyOrg := *o
		copyOrg.Documents = r.documentsForOrgLocked(copyOrg.ID)
		out = append(out, copyOrg)
	}

	return admin.PaginatedResponse[admin.Organization]{
		Data: out,
		Meta: admin.PaginationMeta{
			Total:      len(out),
			Page:       1,
			Limit:      len(out),
			TotalPages: 1,
		},
	}, nil
}

func (r *memoryAdminRepo) GetOrganization(ctx context.Context, id string) (*admin.Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	o, ok := r.orgs[id]
	if !ok {
		return nil, admin.ErrNotFound
	}
	copyOrg := *o
	copyOrg.Documents = r.documentsForOrgLocked(copyOrg.ID)
	return &copyOrg, nil
}

func (r *memoryAdminRepo) UpdateOrganizationStatus(ctx context.Context, id string, status types.AccountStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	o, ok := r.orgs[id]
	if !ok {
		return admin.ErrNotFound
	}
	o.Status = status
	return nil
}

func (r *memoryAdminRepo) DeleteOrganization(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.orgs[id]; !ok {
		return admin.ErrNotFound
	}
	delete(r.orgs, id)
	for docID, d := range r.docs {
		if d.OrganizationID == id {
			delete(r.docs, docID)
		}
	}
	return nil
}

func (r *memoryAdminRepo) ListDocuments(ctx context.Context, params admin.ListDocumentsParams) (admin.PaginatedResponse[admin.OrganizationDocument], error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]admin.OrganizationDocument, 0, len(r.docs))
	for _, d := range r.docs {
		if params.OrgID != "" && d.OrganizationID != params.OrgID {
			continue
		}
		out = append(out, *d)
	}

	return admin.PaginatedResponse[admin.OrganizationDocument]{
		Data: out,
		Meta: admin.PaginationMeta{
			Total:      len(out),
			Page:       1,
			Limit:      len(out),
			TotalPages: 1,
		},
	}, nil
}

func (r *memoryAdminRepo) GetDocument(ctx context.Context, id string) (*admin.OrganizationDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.docs[id]
	if !ok {
		return nil, admin.ErrNotFound
	}
	copyDoc := *d
	return &copyDoc, nil
}

func (r *memoryAdminRepo) ApproveDocument(ctx context.Context, id string, reviewerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.docs[id]
	if !ok {
		return admin.ErrNotFound
	}
	now := time.Now()
	d.Status = types.DocumentStatusApproved
	d.ReviewedAt = &now
	d.ReviewedByID = &reviewerID
	d.RejectionReason = nil

	if o, ok := r.orgs[d.OrganizationID]; ok {
		o.VerificationStatus = types.VerificationStatusApproved
	}

	return nil
}

func (r *memoryAdminRepo) RejectDocument(ctx context.Context, id string, reviewerID string, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.docs[id]
	if !ok {
		return admin.ErrNotFound
	}
	now := time.Now()
	d.Status = types.DocumentStatusRejected
	d.ReviewedAt = &now
	d.ReviewedByID = &reviewerID
	d.RejectionReason = &reason

	if o, ok := r.orgs[d.OrganizationID]; ok {
		o.VerificationStatus = types.VerificationStatusRejected
	}

	return nil
}

func (r *memoryAdminRepo) documentsForOrgLocked(orgID string) []admin.OrganizationDocument {
	out := make([]admin.OrganizationDocument, 0)
	for _, d := range r.docs {
		if d.OrganizationID == orgID {
			out = append(out, *d)
		}
	}
	return out
}

type memoryUploadsRepo struct {
	mu      sync.Mutex
	nextID  int
	uploads map[string]*uploads.Upload
}

func newMemoryUploadsRepo() *memoryUploadsRepo {
	return &memoryUploadsRepo{
		uploads: map[string]*uploads.Upload{},
	}
}

func (r *memoryUploadsRepo) Create(ctx context.Context, provider string, storageKey string, downloadURL *string, fileName string, fileSize int64, mimeType string) (*uploads.Upload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	id := fmt.Sprintf("up_%d", r.nextID)
	now := time.Now()
	u := &uploads.Upload{
		ID:          id,
		Provider:    provider,
		StorageKey:  storageKey,
		DownloadURL: downloadURL,
		FileName:    fileName,
		FileSize:    fileSize,
		MimeType:    mimeType,
		Status:      uploads.UploadStatusPending,
		CreatedAt:   now,
	}
	r.uploads[id] = u
	return u, nil
}

func (r *memoryUploadsRepo) MarkCompleted(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.uploads[id]
	if !ok {
		return uploads.ErrNotFound
	}
	now := time.Now()
	u.Status = uploads.UploadStatusCompleted
	u.CompletedAt = &now
	return nil
}

func (r *memoryUploadsRepo) Get(ctx context.Context, id string) (*uploads.Upload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.uploads[id]
	if !ok {
		return nil, uploads.ErrNotFound
	}
	return u, nil
}

type fakePresigner struct{}

func (p fakePresigner) PresignPutObject(ctx context.Context, bucket string, key string, contentType string, contentLength int64, expiry time.Duration) (storage.PresignResult, error) {
	return storage.PresignResult{
		URL:     "https://example.com/" + bucket + "/" + key,
		Method:  "PUT",
		Headers: storage.DefaultHeaders(contentType, contentLength),
		Expires: time.Now().Add(expiry),
	}, nil
}

func (p fakePresigner) PresignGetObject(ctx context.Context, bucket string, key string, expiry time.Duration) (storage.PresignResult, error) {
	return storage.PresignResult{
		URL:     "https://example.com/" + bucket + "/" + key + "?download=1",
		Method:  "GET",
		Headers: map[string]string{},
		Expires: time.Now().Add(expiry),
	}, nil
}

type memoryOrgRepo struct {
	mu        sync.Mutex
	nextOrgID int
	nextDocID int

	authRepo    *memoryAuthRepo
	adminRepo   *memoryAdminRepo
	uploadsRepo *memoryUploadsRepo
}

func newMemoryOrgRepo(authRepo *memoryAuthRepo, adminRepo *memoryAdminRepo, uploadsRepo *memoryUploadsRepo) *memoryOrgRepo {
	return &memoryOrgRepo{
		authRepo:    authRepo,
		adminRepo:   adminRepo,
		uploadsRepo: uploadsRepo,
	}
}

func (r *memoryOrgRepo) SignupOrganization(ctx context.Context, in org.SignupOrganizationInput) (*org.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerQKID, err := types.NormalizeQKID(in.OwnerQKID)
	if err != nil {
		return nil, err
	}

	bu, err := r.uploadsRepo.Get(ctx, in.BusinessCertificateUploadID)
	if err != nil || bu.Status != uploads.UploadStatusCompleted {
		return nil, org.ErrInvalidUpload
	}
	for _, reg := range in.RegistrantIDs {
		u, err := r.uploadsRepo.Get(ctx, reg.IDDocumentUploadID)
		if err != nil || u.Status != uploads.UploadStatusCompleted {
			return nil, org.ErrInvalidUpload
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.OwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := r.authRepo.CreateUser(ctx, ownerQKID, nil, string(hash), "user")
	if err != nil {
		return nil, err
	}

	r.nextOrgID++
	orgID := fmt.Sprintf("org_%d", r.nextOrgID)
	now := time.Now()

	r.adminRepo.mu.Lock()
	r.adminRepo.orgs[orgID] = &admin.Organization{
		ID:                 orgID,
		Name:               in.OrganizationName,
		Email:              in.OrganizationEmail,
		Phone:              in.OrganizationPhone,
		Status:             types.AccountStatusActive,
		VerificationStatus: types.VerificationStatusPending,
		MemberCount:        1,
		CreatedAt:          now,
		Documents:          nil,
	}
	r.adminRepo.mu.Unlock()

	r.nextDocID++
	businessDocID := fmt.Sprintf("doc_%d", r.nextDocID)
	r.adminRepo.mu.Lock()
	r.adminRepo.docs[businessDocID] = &admin.OrganizationDocument{
		ID:               businessDocID,
		OrganizationID:   orgID,
		OrganizationName: in.OrganizationName,
		Type:             types.DocumentTypeRegistrationCertificate,
		StorageKey:       bu.StorageKey,
		FileName:         bu.FileName,
		FileSize:         bu.FileSize,
		MimeType:         bu.MimeType,
		DownloadURL:      bu.StorageKey,
		Status:           types.DocumentStatusPending,
		UploadedAt:       now,
	}
	r.adminRepo.mu.Unlock()

	for range in.RegistrantIDs {
		r.nextDocID++
		docID := fmt.Sprintf("doc_%d", r.nextDocID)
		r.adminRepo.mu.Lock()
		r.adminRepo.docs[docID] = &admin.OrganizationDocument{
			ID:               docID,
			OrganizationID:   orgID,
			OrganizationName: in.OrganizationName,
			Type:             types.DocumentTypeIDDocument,
			StorageKey:       "uploads/mock/" + docID,
			FileName:         "id.png",
			FileSize:         10,
			MimeType:         "image/png",
			DownloadURL:      "s3://mock",
			Status:           types.DocumentStatusPending,
			UploadedAt:       now,
		}
		r.adminRepo.mu.Unlock()
	}

	return &org.Result{
		UserID:             user.ID,
		OrganizationID:     orgID,
		VerificationStatus: types.VerificationStatusPending,
	}, nil
}

func newTestRouter(t *testing.T) (*memoryAuthRepo, *memoryAdminRepo, http.Handler) {
	t.Setenv("BOOTSTRAP_ADMIN_QKID", "")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "")

	cfg := config.Config{
		Port:               "8080",
		AppEnv:             "test",
		JWTAccessSecret:    "test-access-secret",
		JWTRefreshSecret:   "test-refresh-secret",
		CookieDomain:       "",
		CORSAllowedOrigins: "",
		S3Bucket:           "test-bucket",
	}

	repo := newMemoryAuthRepo()
	adminRepo := newMemoryAdminRepo(repo)
	uploadsRepo := newMemoryUploadsRepo()
	orgRepo := newMemoryOrgRepo(repo, adminRepo, uploadsRepo)
	router := NewRouter(cfg, repo, adminRepo, uploadsRepo, fakePresigner{}, orgRepo, zap.NewNop())
	return repo, adminRepo, router
}

func doJSON(t *testing.T, h http.Handler, method string, path string, payload any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1234"
	for _, c := range cookies {
		req.AddCookie(c)
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func findCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestAPIEndpoints(t *testing.T) {
	repo, adminRepo, router := newTestRouter(t)

	t.Run("healthz", func(t *testing.T) {
		rr := doJSON(t, router, http.MethodGet, "/healthz", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("signup and login", func(t *testing.T) {
		signupRR := doJSON(t, router, http.MethodPost, "/signup", map[string]any{
			"qkId":     "bob",
			"password": "password123",
		})
		if signupRR.Code != http.StatusCreated {
			t.Fatalf("status = %d, body=%s", signupRR.Code, signupRR.Body.String())
		}

		var signupRes struct {
			User  auth.User `json:"user"`
			Token string    `json:"token"`
		}
		if err := json.Unmarshal(signupRR.Body.Bytes(), &signupRes); err != nil {
			t.Fatalf("unmarshal signup: %v", err)
		}
		if signupRes.User.QKID != "bob.qk" {
			t.Fatalf("qkId = %q", signupRes.User.QKID)
		}
		if signupRes.Token == "" {
			t.Fatalf("token is empty")
		}

		loginRR := doJSON(t, router, http.MethodPost, "/login", map[string]any{
			"qkId":     "bob.qk",
			"password": "password123",
		})
		if loginRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", loginRR.Code, loginRR.Body.String())
		}
	})

	t.Run("admin auth flow", func(t *testing.T) {
		adminHash, err := bcrypt.GenerateFromPassword([]byte("adminpass123"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
		if _, err := repo.CreateUser(context.Background(), "superadmin.qk", nil, string(adminHash), "admin"); err != nil {
			t.Fatalf("create admin user: %v", err)
		}

		meNoCookieRR := doJSON(t, router, http.MethodGet, "/admin/auth/me", nil)
		if meNoCookieRR.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, body=%s", meNoCookieRR.Code, meNoCookieRR.Body.String())
		}

		loginRR := doJSON(t, router, http.MethodPost, "/admin/auth/login", map[string]any{
			"qkId":     "superadmin.qk",
			"password": "adminpass123",
		})
		if loginRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", loginRR.Code, loginRR.Body.String())
		}

		loginResp := loginRR.Result()
		accessCookie := findCookie(loginResp, "access_token")
		refreshCookie := findCookie(loginResp, "refresh_token")
		if accessCookie == nil || accessCookie.Value == "" {
			t.Fatalf("missing access_token cookie")
		}
		if refreshCookie == nil || refreshCookie.Value == "" {
			t.Fatalf("missing refresh_token cookie")
		}

		meRR := doJSON(t, router, http.MethodGet, "/admin/auth/me", nil, accessCookie)
		if meRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", meRR.Code, meRR.Body.String())
		}

		refreshRR := doJSON(t, router, http.MethodPost, "/admin/auth/refresh", nil, refreshCookie)
		if refreshRR.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body=%s", refreshRR.Code, refreshRR.Body.String())
		}
		refreshResp := refreshRR.Result()
		newAccessCookie := findCookie(refreshResp, "access_token")
		newRefreshCookie := findCookie(refreshResp, "refresh_token")
		if newAccessCookie == nil || newAccessCookie.Value == "" {
			t.Fatalf("missing new access_token cookie")
		}
		if newRefreshCookie == nil || newRefreshCookie.Value == "" {
			t.Fatalf("missing new refresh_token cookie")
		}
		if newRefreshCookie.Value == refreshCookie.Value {
			t.Fatalf("refresh token did not rotate")
		}

		refreshOldRR := doJSON(t, router, http.MethodPost, "/admin/auth/refresh", nil, refreshCookie)
		if refreshOldRR.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, body=%s", refreshOldRR.Code, refreshOldRR.Body.String())
		}

		logoutRR := doJSON(t, router, http.MethodPost, "/admin/auth/logout", nil, newRefreshCookie)
		if logoutRR.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body=%s", logoutRR.Code, logoutRR.Body.String())
		}

		logoutResp := logoutRR.Result()
		clearedAccess := findCookie(logoutResp, "access_token")
		clearedRefresh := findCookie(logoutResp, "refresh_token")
		if clearedAccess == nil || clearedAccess.MaxAge >= 0 {
			t.Fatalf("access_token was not cleared")
		}
		if clearedRefresh == nil || clearedRefresh.MaxAge >= 0 {
			t.Fatalf("refresh_token was not cleared")
		}
	})

	t.Run("admin resources", func(t *testing.T) {
		adminHash, err := bcrypt.GenerateFromPassword([]byte("adminpass123"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
		adminUser, err := repo.CreateUser(context.Background(), "superadmin.qk", nil, string(adminHash), "admin")
		if err != nil {
			if err == auth.ErrIdentityTaken {
				adminUser, err = repo.GetUserByQKID(context.Background(), "superadmin.qk")
			}
			if err != nil {
				t.Fatalf("create admin user: %v", err)
			}
		}

		loginRR := doJSON(t, router, http.MethodPost, "/admin/auth/login", map[string]any{
			"qkId":     "superadmin.qk",
			"password": "adminpass123",
		})
		if loginRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", loginRR.Code, loginRR.Body.String())
		}
		accessCookie := findCookie(loginRR.Result(), "access_token")
		if accessCookie == nil {
			t.Fatalf("missing access cookie")
		}

		signupRR := doJSON(t, router, http.MethodPost, "/signup", map[string]any{
			"qkId":     "bob2",
			"password": "password123",
		})
		if signupRR.Code != http.StatusCreated {
			t.Fatalf("status = %d, body=%s", signupRR.Code, signupRR.Body.String())
		}

		usersRR := doJSON(t, router, http.MethodGet, "/admin/users?page=1&limit=50", nil, accessCookie)
		if usersRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", usersRR.Code, usersRR.Body.String())
		}

		var usersRes admin.PaginatedResponse[admin.AdminListUser]
		if err := json.Unmarshal(usersRR.Body.Bytes(), &usersRes); err != nil {
			t.Fatalf("unmarshal users: %v", err)
		}
		if usersRes.Meta.Total < 1 {
			t.Fatalf("expected users in response")
		}

		var targetUserID string
		for _, u := range usersRes.Data {
			if u.Email == nil && u.Status == types.AccountStatusActive {
				targetUserID = u.ID
				break
			}
		}
		if targetUserID == "" {
			t.Fatalf("could not find target user")
		}

		suspendRR := doJSON(t, router, http.MethodPatch, "/admin/users/"+targetUserID+"/suspend", map[string]any{}, accessCookie)
		if suspendRR.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body=%s", suspendRR.Code, suspendRR.Body.String())
		}

		userRR := doJSON(t, router, http.MethodGet, "/admin/users/"+targetUserID, nil, accessCookie)
		if userRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", userRR.Code, userRR.Body.String())
		}

		var userRes admin.AdminListUser
		if err := json.Unmarshal(userRR.Body.Bytes(), &userRes); err != nil {
			t.Fatalf("unmarshal user: %v", err)
		}
		if userRes.Status != types.AccountStatusSuspended {
			t.Fatalf("expected suspended status, got %q", userRes.Status)
		}

		adminRepo.mu.Lock()
		org := &admin.Organization{
			ID:                 "org_1",
			Name:               "Acme Inc",
			Status:             types.AccountStatusActive,
			VerificationStatus: types.VerificationStatusPending,
			MemberCount:        0,
			CreatedAt:          time.Now(),
			Documents:          nil,
		}
		adminRepo.orgs[org.ID] = org
		doc := &admin.OrganizationDocument{
			ID:               "doc_1",
			OrganizationID:   org.ID,
			OrganizationName: org.Name,
			Type:             types.DocumentTypeRegistrationCertificate,
			StorageKey:       "uploads/doc_1/reg.pdf",
			FileName:         "reg.pdf",
			FileSize:         123,
			MimeType:         "application/pdf",
			DownloadURL:      "s3://bucket/key",
			Status:           types.DocumentStatusPending,
			UploadedAt:       time.Now(),
		}
		adminRepo.docs[doc.ID] = doc
		adminRepo.mu.Unlock()

		orgsRR := doJSON(t, router, http.MethodGet, "/admin/organizations?page=1&limit=50", nil, accessCookie)
		if orgsRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", orgsRR.Code, orgsRR.Body.String())
		}

		docsRR := doJSON(t, router, http.MethodGet, "/admin/documents?page=1&limit=50", nil, accessCookie)
		if docsRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", docsRR.Code, docsRR.Body.String())
		}

		approveRR := doJSON(t, router, http.MethodPatch, "/admin/documents/doc_1/approve", nil, accessCookie)
		if approveRR.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body=%s", approveRR.Code, approveRR.Body.String())
		}

		docRR := doJSON(t, router, http.MethodGet, "/admin/documents/doc_1", nil, accessCookie)
		if docRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", docRR.Code, docRR.Body.String())
		}
		var docRes admin.OrganizationDocument
		if err := json.Unmarshal(docRR.Body.Bytes(), &docRes); err != nil {
			t.Fatalf("unmarshal doc: %v", err)
		}
		if docRes.Status != types.DocumentStatusApproved {
			t.Fatalf("expected approved status, got %q", docRes.Status)
		}
		if docRes.ReviewedByID == nil || *docRes.ReviewedByID != adminUser.ID {
			t.Fatalf("expected reviewedById = %q", adminUser.ID)
		}

		downloadRR := doJSON(t, router, http.MethodGet, "/admin/documents/doc_1/download", nil, accessCookie)
		if downloadRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", downloadRR.Code, downloadRR.Body.String())
		}
		var downloadRes struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(downloadRR.Body.Bytes(), &downloadRes); err != nil {
			t.Fatalf("unmarshal download: %v", err)
		}
		if downloadRes.URL == "" {
			t.Fatalf("expected non-empty url")
		}
	})

	t.Run("uploads and organization signup", func(t *testing.T) {
		adminHash, err := bcrypt.GenerateFromPassword([]byte("adminpass123"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
		_, err = repo.CreateUser(context.Background(), "superadmin.qk", nil, string(adminHash), "admin")
		if err != nil && err != auth.ErrIdentityTaken {
			t.Fatalf("create admin user: %v", err)
		}

		loginRR := doJSON(t, router, http.MethodPost, "/admin/auth/login", map[string]any{
			"qkId":     "superadmin.qk",
			"password": "adminpass123",
		})
		if loginRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", loginRR.Code, loginRR.Body.String())
		}
		accessCookie := findCookie(loginRR.Result(), "access_token")
		if accessCookie == nil {
			t.Fatalf("missing access cookie")
		}

		presignBusinessRR := doJSON(t, router, http.MethodPost, "/uploads/presign", map[string]any{
			"fileName": "cert.pdf",
			"fileSize": 10,
			"mimeType": "application/pdf",
		})
		if presignBusinessRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", presignBusinessRR.Code, presignBusinessRR.Body.String())
		}
		var presignBusiness struct {
			UploadID string `json:"uploadId"`
		}
		if err := json.Unmarshal(presignBusinessRR.Body.Bytes(), &presignBusiness); err != nil {
			t.Fatalf("unmarshal presign business: %v", err)
		}

		presignRegistrantRR := doJSON(t, router, http.MethodPost, "/uploads/presign", map[string]any{
			"fileName": "id.png",
			"fileSize": 10,
			"mimeType": "image/png",
		})
		if presignRegistrantRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", presignRegistrantRR.Code, presignRegistrantRR.Body.String())
		}
		var presignRegistrant struct {
			UploadID string `json:"uploadId"`
		}
		if err := json.Unmarshal(presignRegistrantRR.Body.Bytes(), &presignRegistrant); err != nil {
			t.Fatalf("unmarshal presign registrant: %v", err)
		}

		completeBusinessRR := doJSON(t, router, http.MethodPost, "/uploads/complete", map[string]any{
			"uploadId": presignBusiness.UploadID,
		})
		if completeBusinessRR.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body=%s", completeBusinessRR.Code, completeBusinessRR.Body.String())
		}

		completeRegistrantRR := doJSON(t, router, http.MethodPost, "/uploads/complete", map[string]any{
			"uploadId": presignRegistrant.UploadID,
		})
		if completeRegistrantRR.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body=%s", completeRegistrantRR.Code, completeRegistrantRR.Body.String())
		}

		orgSignupRR := doJSON(t, router, http.MethodPost, "/signup/organization", map[string]any{
			"qkId":                        "orgowner",
			"password":                    "password123",
			"organizationName":            "Org Co",
			"businessCertificateUploadId": presignBusiness.UploadID,
			"registrants": []map[string]any{
				{
					"fullLegalName":      "Alice Admin",
					"idDocumentUploadId": presignRegistrant.UploadID,
				},
			},
		})
		if orgSignupRR.Code != http.StatusCreated {
			t.Fatalf("status = %d, body=%s", orgSignupRR.Code, orgSignupRR.Body.String())
		}

		orgsRR := doJSON(t, router, http.MethodGet, "/admin/organizations?page=1&limit=50", nil, accessCookie)
		if orgsRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", orgsRR.Code, orgsRR.Body.String())
		}
		var orgsRes admin.PaginatedResponse[admin.Organization]
		if err := json.Unmarshal(orgsRR.Body.Bytes(), &orgsRes); err != nil {
			t.Fatalf("unmarshal orgs: %v", err)
		}
		if orgsRes.Meta.Total < 1 {
			t.Fatalf("expected orgs in response")
		}

		docsRR := doJSON(t, router, http.MethodGet, "/admin/documents?page=1&limit=50", nil, accessCookie)
		if docsRR.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", docsRR.Code, docsRR.Body.String())
		}
		var docsRes admin.PaginatedResponse[admin.OrganizationDocument]
		if err := json.Unmarshal(docsRR.Body.Bytes(), &docsRes); err != nil {
			t.Fatalf("unmarshal docs: %v", err)
		}
		if docsRes.Meta.Total < 1 {
			t.Fatalf("expected docs in response")
		}
	})
}
