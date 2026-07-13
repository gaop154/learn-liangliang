package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"learn-liangliang/backend/internal/config"
	"learn-liangliang/backend/internal/db"
)

type failingSessionStore struct{}

func (failingSessionStore) GetUserByUsername(context.Context, string) (db.User, error) {
	return db.User{}, errors.New("不应调用用户查询")
}

func (failingSessionStore) CreateSession(context.Context, int64, string, time.Time) error {
	return errors.New("不应创建会话")
}

func (failingSessionStore) GetActiveSessionByTokenHash(context.Context, string) (db.SessionUser, error) {
	return db.SessionUser{}, errors.New("数据库不可用")
}

func (failingSessionStore) RevokeSession(context.Context, string) error {
	return errors.New("不应撤销会话")
}

func TestInternalAuthenticateRejectsAnonymousRequest(t *testing.T) {
	handler := &Handler{}
	request := httptest.NewRequest(http.MethodGet, "/internal/authenticate", nil)
	response := httptest.NewRecorder()

	handler.InternalAuthenticate(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("InternalAuthenticate() status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("InternalAuthenticate() body = %q, want empty body", response.Body.String())
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
	}
}

func TestInternalAuthenticateAcceptsAttachedUser(t *testing.T) {
	handler := &Handler{}
	request := httptest.NewRequest(http.MethodGet, "/internal/authenticate", nil)
	request = request.WithContext(context.WithValue(request.Context(), userContextKey, UserPayload{ID: 1, Username: "admin"}))
	response := httptest.NewRecorder()

	handler.InternalAuthenticate(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("InternalAuthenticate() status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("InternalAuthenticate() body = %q, want empty body", response.Body.String())
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
	}
}

func TestAttachUserFailsClosedForInternalAuthenticateOnSessionLookupError(t *testing.T) {
	handler := NewHandler(failingSessionStore{}, config.Config{CookieName: "session"})
	nextCalled := false
	protected := handler.AttachUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		handler.InternalAuthenticate(w, r)
	}))
	request := httptest.NewRequest(http.MethodGet, "/internal/authenticate", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "token"})
	response := httptest.NewRecorder()

	protected.ServeHTTP(response, request)

	if nextCalled {
		t.Fatal("会话查询失败时不应继续执行内部认证处理器")
	}
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("AttachUser() status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("AttachUser() body = %q, want empty body", response.Body.String())
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
	}
}
