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

	"qwikkle-api/internal/auth"
	"qwikkle-api/internal/config"
	"qwikkle-api/internal/types"
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

func newTestRouter(t *testing.T) (*memoryAuthRepo, http.Handler) {
	t.Setenv("BOOTSTRAP_ADMIN_QKID", "")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "")

	cfg := config.Config{
		Port:               "8080",
		AppEnv:             "test",
		JWTAccessSecret:    "test-access-secret",
		JWTRefreshSecret:   "test-refresh-secret",
		CookieDomain:       "",
		CORSAllowedOrigins: "",
	}

	repo := newMemoryAuthRepo()
	router := NewRouter(cfg, repo, zap.NewNop())
	return repo, router
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
	repo, router := newTestRouter(t)

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
}
