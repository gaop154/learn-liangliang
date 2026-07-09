package reading

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"learn-liangliang/backend/internal/auth"
	"learn-liangliang/backend/internal/db"
	"learn-liangliang/backend/internal/response"
)

type Handler struct {
	store *db.Store
}

type upsertRequest struct {
	ArticlePath     string `json:"articlePath"`
	ArticleTitle    string `json:"articleTitle"`
	ProgressPercent int    `json:"progressPercent"`
	ScrollY         int    `json:"scrollY"`
	Finished        bool   `json:"finished"`
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	current, ok := auth.CurrentUser(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req upsertRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "请求格式不正确")
		return
	}
	if err := validateRequest(req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	progress, err := h.store.UpsertReadingProgress(r.Context(), current.ID, req.ArticlePath, req.ArticleTitle, req.ProgressPercent, req.ScrollY, req.Finished)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SAVE_PROGRESS_FAILED", "保存阅读进度失败")
		return
	}
	response.JSON(w, http.StatusOK, progress)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	current, ok := auth.CurrentUser(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
		return
	}

	articlePath := strings.TrimSpace(r.URL.Query().Get("articlePath"))
	if articlePath == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "articlePath 不能为空")
		return
	}

	progress, err := h.store.GetReadingProgress(r.Context(), current.ID, articlePath)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			response.JSON(w, http.StatusOK, map[string]any{"found": false})
			return
		}
		response.Error(w, http.StatusInternalServerError, "GET_PROGRESS_FAILED", "查询阅读进度失败")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"found": true, "item": progress})
}

func (h *Handler) Recent(w http.ResponseWriter, r *http.Request) {
	current, ok := auth.CurrentUser(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
		return
	}

	limit := int32(20)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "limit 必须是正整数")
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = int32(parsed)
	}

	items, err := h.store.ListRecentReadingProgress(r.Context(), current.ID, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_PROGRESS_FAILED", "查询最近阅读失败")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func validateRequest(req upsertRequest) error {
	if strings.TrimSpace(req.ArticlePath) == "" {
		return errors.New("articlePath 不能为空")
	}
	if !strings.HasPrefix(req.ArticlePath, "/") || strings.HasPrefix(req.ArticlePath, "//") || strings.Contains(req.ArticlePath, "..") || len(req.ArticlePath) > 1024 {
		return errors.New("articlePath 不合法")
	}
	if strings.TrimSpace(req.ArticleTitle) == "" {
		return errors.New("articleTitle 不能为空")
	}
	if len(req.ArticleTitle) > 500 {
		return errors.New("articleTitle 不能超过 500 个字符")
	}
	if req.ProgressPercent < 0 || req.ProgressPercent > 100 {
		return errors.New("progressPercent 必须在 0 到 100 之间")
	}
	if req.ScrollY < 0 {
		return errors.New("scrollY 不能小于 0")
	}
	return nil
}
