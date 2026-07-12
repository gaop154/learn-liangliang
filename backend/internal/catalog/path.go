package catalog

import (
	"errors"
	"net/url"
	"path"
	"strings"
)

const MaxPathLength = 1024

type ContentType string

const (
	ContentTypeCourseArticle     ContentType = "course_article"
	ContentTypeArticle           ContentType = "article"
	ContentTypeGeektimeArticle   ContentType = "geektime_article"
	ContentTypeLoveCourseArticle ContentType = "love_course_article"
	ContentTypePDF               ContentType = "pdf"
)

func CanonicalizePath(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("路径不能为空")
	}
	if len(value) > MaxPathLength {
		return "", errors.New("路径不合法")
	}

	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", errors.New("路径不合法")
	}
	value = strings.ReplaceAll(decoded, "\\", "/")
	if strings.Contains(value, "\x00") || len(value) > MaxPathLength {
		return "", errors.New("路径不合法")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", errors.New("路径不合法")
		}
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = path.Clean(value)
	if value == "." || value == "/" || strings.HasPrefix(value, "//") {
		return "", errors.New("路径不合法")
	}
	if strings.HasPrefix(value, "/content/") {
		value = strings.TrimPrefix(value, "/content")
	} else if value == "/content" {
		return "", errors.New("路径不合法")
	}
	if !strings.HasPrefix(value, "/") || strings.Contains(value, "/../") || strings.HasSuffix(value, "/..") {
		return "", errors.New("路径不合法")
	}
	return value, nil
}

func ArticleContentType(articlePath string) (ContentType, bool) {
	if !strings.HasSuffix(articlePath, ".md.html") {
		return "", false
	}
	segments := strings.Split(strings.TrimPrefix(articlePath, "/"), "/")
	if len(segments) != 3 || segments[1] == "" || segments[2] == "" {
		return "", false
	}

	switch segments[0] {
	case "专栏":
		return ContentTypeCourseArticle, true
	case "其他":
		switch segments[1] {
		case "恋爱必修课":
			return ContentTypeLoveCourseArticle, true
		case "文章":
			return ContentTypeArticle, true
		case "极客时间":
			return ContentTypeGeektimeArticle, true
		}
	}
	return "", false
}

func IsArticlePath(articlePath string) bool {
	_, ok := ArticleContentType(articlePath)
	return ok
}

func IsCoursePath(coursePath string) bool {
	segments := strings.Split(strings.TrimPrefix(coursePath, "/"), "/")
	return len(segments) == 2 && segments[0] == "专栏" && segments[1] != ""
}
