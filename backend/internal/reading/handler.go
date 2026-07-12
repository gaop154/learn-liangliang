package reading

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"learn-liangliang/backend/internal/auth"
	"learn-liangliang/backend/internal/catalog"
	"learn-liangliang/backend/internal/db"
	"learn-liangliang/backend/internal/response"
)

const maxBatchPaths = 500

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

type articlePathsRequest struct {
	ArticlePaths []string `json:"articlePaths"`
}

type coursePathsRequest struct {
	CoursePaths []string `json:"coursePaths"`
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

	var req upsertRequest
	if err := decodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "请求格式不正确")
		return
	}
	canonicalPath, err := canonicalizeArticlePath(req.ArticlePath)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	req.ArticlePath = canonicalPath
	if err := validateRequest(req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	progress, err := h.store.UpsertReadingProgress(r.Context(), current.ID, req.ArticlePath, req.ArticleTitle, req.ProgressPercent, req.ScrollY, req.Finished)
	if errors.Is(err, db.ErrNotFound) {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "articlePath 不是活动文章")
		return
	}
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

	articlePath, err := canonicalizeArticlePath(r.URL.Query().Get("articlePath"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if !isArticlePath(articlePath) {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "articlePath 必须以 .md.html 结尾")
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

func (h *Handler) Batch(w http.ResponseWriter, r *http.Request) {
	current, ok := auth.CurrentUser(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
		return
	}

	var req articlePathsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "请求格式不正确")
		return
	}
	articlePaths, err := canonicalizeArticlePaths(req.ArticlePaths)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	items, err := h.store.ListReadingProgressByArticlePaths(r.Context(), current.ID, articlePaths)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_PROGRESS_FAILED", "查询章节进度失败")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CourseResumes(w http.ResponseWriter, r *http.Request) {
	current, ok := auth.CurrentUser(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
		return
	}

	var req coursePathsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "请求格式不正确")
		return
	}
	coursePaths, err := canonicalizeCoursePaths(req.CoursePaths)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	items, err := h.store.ListLatestReadingProgressByCoursePaths(r.Context(), current.ID, coursePaths)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_PROGRESS_FAILED", "查询课程继续阅读失败")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CourseSummaries(w http.ResponseWriter, r *http.Request) {
	current, ok := auth.CurrentUser(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
		return
	}
	items, err := h.store.ListCourseSummaries(r.Context(), current.ID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_COURSES_FAILED", "查询课程进度失败")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CourseDetail(w http.ResponseWriter, r *http.Request) {
	current, ok := auth.CurrentUser(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "请先登录")
		return
	}
	coursePath, err := canonicalizeArticlePath(r.URL.Query().Get("coursePath"))
	if err != nil || !isCoursePath(coursePath) {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "coursePath 不合法")
		return
	}
	summary, articles, err := h.store.GetCourseSummary(r.Context(), current.ID, coursePath)
	if errors.Is(err, db.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "课程不存在或已下线")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "GET_COURSE_FAILED", "查询课程进度失败")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"course": summary, "articles": articles})
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

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("请求格式不正确")
	}
	return nil
}

func canonicalizeArticlePaths(rawPaths []string) ([]string, error) {
	if len(rawPaths) == 0 || len(rawPaths) > maxBatchPaths {
		return nil, errors.New("articlePaths 数量必须在 1 到 500 之间")
	}
	paths := make([]string, 0, len(rawPaths))
	seen := make(map[string]struct{}, len(rawPaths))
	for _, rawPath := range rawPaths {
		articlePath, err := canonicalizeArticlePath(rawPath)
		if err != nil {
			return nil, err
		}
		if !isArticlePath(articlePath) {
			return nil, errors.New("articlePath 必须以 .md.html 结尾")
		}
		if _, exists := seen[articlePath]; exists {
			return nil, errors.New("articlePaths 不能包含重复路径")
		}
		seen[articlePath] = struct{}{}
		paths = append(paths, articlePath)
	}
	return paths, nil
}

func canonicalizeCoursePaths(rawPaths []string) ([]string, error) {
	if len(rawPaths) == 0 || len(rawPaths) > maxBatchPaths {
		return nil, errors.New("coursePaths 数量必须在 1 到 500 之间")
	}
	paths := make([]string, 0, len(rawPaths))
	seen := make(map[string]struct{}, len(rawPaths))
	for _, rawPath := range rawPaths {
		coursePath, err := canonicalizeArticlePath(rawPath)
		if err != nil {
			return nil, errors.New("coursePath 不合法")
		}
		if !isCoursePath(coursePath) {
			return nil, errors.New("coursePath 不合法")
		}
		if _, exists := seen[coursePath]; exists {
			return nil, errors.New("coursePaths 不能包含重复路径")
		}
		seen[coursePath] = struct{}{}
		paths = append(paths, coursePath)
	}
	return paths, nil
}

func canonicalizeArticlePath(raw string) (string, error) {
	articlePath, err := catalog.CanonicalizePath(raw)
	if err != nil {
		if strings.TrimSpace(raw) == "" {
			return "", errors.New("articlePath 不能为空")
		}
		return "", errors.New("articlePath 不合法")
	}
	return articlePath, nil
}

func isArticlePath(articlePath string) bool {
	return catalog.IsArticlePath(articlePath)
}

func isCoursePath(coursePath string) bool {
	return catalog.IsCoursePath(coursePath)
}

func validateRequest(req upsertRequest) error {
	if strings.TrimSpace(req.ArticlePath) == "" {
		return errors.New("articlePath 不能为空")
	}
	if !strings.HasPrefix(req.ArticlePath, "/") || strings.HasPrefix(req.ArticlePath, "//") || len(req.ArticlePath) > 1024 {
		return errors.New("articlePath 不合法")
	}
	if !isArticlePath(req.ArticlePath) {
		return errors.New("articlePath 必须以 .md.html 结尾")
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
	if req.Finished && req.ProgressPercent != 100 {
		return errors.New("finished 仅可在 progressPercent 为 100 时设置")
	}
	return nil
}
