package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"learn-liangliang/backend/internal/config"
	"learn-liangliang/backend/internal/db"
	"learn-liangliang/backend/internal/response"
)

type contextKey string

const userContextKey contextKey = "user"

// Store 定义认证处理器所需的持久化操作，便于在不连接数据库的情况下验证
// 会话校验异常必须拒绝受保护内容。
type Store interface {
	GetUserByUsername(ctx context.Context, username string) (db.User, error)
	CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	GetActiveSessionByTokenHash(ctx context.Context, tokenHash string) (db.SessionUser, error)
	RevokeSession(ctx context.Context, tokenHash string) error
}

type Handler struct {
	store Store
	cfg   config.Config
}

type UserPayload struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewHandler(store Store, cfg config.Config) *Handler {
	return &Handler{store: store, cfg: cfg}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req loginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "请求格式不正确")
		return
	}
	if req.Username == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "用户名和密码不能为空")
		return
	}
	if len(req.Username) > 64 || len(req.Password) > 256 {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "用户名或密码长度不合法")
		return
	}

	user, err := h.store.GetUserByUsername(r.Context(), req.Username)
	if err != nil || !user.IsActive || !VerifyPassword(req.Password, user.PasswordHash) {
		response.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名或密码错误")
		return
	}

	token, tokenHash, err := NewSessionToken()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "创建登录会话失败")
		return
	}
	expiresAt := time.Now().Add(h.cfg.SessionTTL)
	if err := h.store.CreateSession(r.Context(), user.ID, tokenHash, expiresAt); err != nil {
		response.Error(w, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "创建登录会话失败")
		return
	}

	h.setCookie(w, token, expiresAt)
	response.JSON(w, http.StatusOK, map[string]any{"user": toUserPayload(user.ID, user.Username, user.DisplayName)})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	current, ok := CurrentUser(r.Context())
	if !ok {
		response.JSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"authenticated": true, "user": current})
}

// InternalAuthenticate 仅供 Nginx auth_request 内部子请求使用。它返回空响应，
// 由静态网关而非 API 决定如何处理未认证的浏览器请求。
func (h *Handler) InternalAuthenticate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if _, ok := CurrentUser(r.Context()); !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.cfg.CookieName)
	if err == nil && cookie.Value != "" {
		_ = h.store.RevokeSession(r.Context(), HashToken(cookie.Value))
	}

	h.clearCookie(w)
	response.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := CurrentUser(r.Context()); !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) AttachUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(h.cfg.CookieName)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		session, err := h.store.GetActiveSessionByTokenHash(r.Context(), HashToken(cookie.Value))
		if err != nil {
			if !errors.Is(err, db.ErrNotFound) {
				if r.URL.Path == "/internal/authenticate" {
					w.Header().Set("Cache-Control", "no-store")
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					response.Error(w, http.StatusInternalServerError, "SESSION_CHECK_FAILED", "校验登录状态失败")
				}
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		current := toUserPayload(session.UserID, session.Username, session.DisplayName)
		ctx := context.WithValue(r.Context(), userContextKey, current)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CurrentUser(ctx context.Context) (UserPayload, bool) {
	current, ok := ctx.Value(userContextKey).(UserPayload)
	return current, ok
}

func (h *Handler) setCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func toUserPayload(id int64, username string, displayName string) UserPayload {
	name := username
	if displayName != "" {
		name = displayName
	}
	return UserPayload{ID: id, Username: username, DisplayName: name}
}
