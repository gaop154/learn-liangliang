package catalog

import (
	"fmt"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type SyncItem struct {
	PublicPath  string
	Title       string
	ContentType ContentType
}

type SyncCourse struct {
	PublicPath string
	Title      string
}

type SyncCourseArticle struct {
	CoursePath  string
	ArticlePath string
	Position    int
}

type SyncSource struct {
	Items          []SyncItem
	Courses        []SyncCourse
	CourseArticles []SyncCourseArticle
}

func BuildSync(root string) (SyncSource, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return SyncSource{}, fmt.Errorf("解析内容目录失败: %w", err)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return SyncSource{}, fmt.Errorf("内容目录不存在或不是目录: %s", root)
	}

	result := SyncSource{}
	courseRoot := filepath.Join(root, "专栏")
	otherRoot := filepath.Join(root, "其他")
	for _, requiredRoot := range []string{courseRoot, otherRoot} {
		info, err := os.Stat(requiredRoot)
		if err != nil || !info.IsDir() {
			return SyncSource{}, fmt.Errorf("缺少内容分类目录: %s", requiredRoot)
		}
	}
	if err := scanCourses(courseRoot, &result); err != nil {
		return SyncSource{}, err
	}
	for _, category := range []struct {
		name string
		typ  ContentType
	}{
		{"恋爱必修课", ContentTypeLoveCourseArticle},
		{"文章", ContentTypeArticle},
		{"极客时间", ContentTypeGeektimeArticle},
	} {
		if err := scanArticles(filepath.Join(root, "其他", category.name), category.typ, &result); err != nil && !os.IsNotExist(err) {
			return SyncSource{}, err
		}
	}
	if err := scanPDFs(filepath.Join(root, "其他", "PDF"), &result); err != nil && !os.IsNotExist(err) {
		return SyncSource{}, err
	}
	return result, nil
}

func scanCourses(courseRoot string, result *SyncSource) error {
	entries, err := os.ReadDir(courseRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "assets" {
			continue
		}
		coursePath := "/专栏/" + entry.Name()
		if !IsCoursePath(coursePath) {
			continue
		}
		indexPath := filepath.Join(courseRoot, entry.Name(), "index.html")
		content, err := os.ReadFile(indexPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("读取课程目录 %s 失败: %w", indexPath, err)
		}
		result.Courses = append(result.Courses, SyncCourse{PublicPath: coursePath, Title: entry.Name()})
		links := BookPostLinks(string(content))
		seen := make(map[string]struct{}, len(links))
		for _, link := range links {
			articlePath, ok := resolveArticleLink(coursePath, link.Href)
			if !ok || !strings.HasPrefix(articlePath, coursePath+"/") {
				continue
			}
			if _, exists := seen[articlePath]; exists {
				continue
			}
			articleFile := filepath.Join(courseRoot, entry.Name(), filepath.Base(articlePath))
			info, err := os.Stat(articleFile)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("检查课程文章 %s 失败: %w", articleFile, err)
			}
			if info.IsDir() {
				continue
			}
			seen[articlePath] = struct{}{}
			title := strings.TrimSpace(link.Text)
			if title == "" {
				title = strings.TrimSuffix(filepath.Base(articlePath), ".html")
			}
			result.Items = append(result.Items, SyncItem{PublicPath: articlePath, Title: title, ContentType: ContentTypeCourseArticle})
			result.CourseArticles = append(result.CourseArticles, SyncCourseArticle{
				CoursePath: coursePath, ArticlePath: articlePath, Position: len(seen) - 1,
			})
		}
	}
	return nil
}

func scanArticles(categoryRoot string, contentType ContentType, result *SyncSource) error {
	indexPath := filepath.Join(categoryRoot, "index.html")
	content, err := os.ReadFile(indexPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取分类目录 %s 失败: %w", indexPath, err)
	}

	seen := make(map[string]struct{})
	for _, link := range BookPostLinks(string(content)) {
		articlePath, ok := resolveArticleLink("/其他/"+filepath.Base(categoryRoot), link.Href)
		if !ok || !strings.HasPrefix(articlePath, "/其他/"+filepath.Base(categoryRoot)+"/") {
			continue
		}
		if _, exists := seen[articlePath]; exists {
			continue
		}
		articleFile := filepath.Join(categoryRoot, filepath.Base(articlePath))
		info, err := os.Stat(articleFile)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("检查分类文章 %s 失败: %w", articleFile, err)
		}
		if info.IsDir() {
			continue
		}
		seen[articlePath] = struct{}{}
		title := strings.TrimSpace(link.Text)
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(articlePath), ".html")
		}
		result.Items = append(result.Items, SyncItem{
			PublicPath: articlePath, Title: title, ContentType: contentType,
		})
	}
	return nil
}

func scanPDFs(pdfRoot string, result *SyncSource) error {
	return filepath.WalkDir(pdfRoot, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".pdf") {
			return nil
		}
		publicPath, err := publicPath(pdfRoot, filePath)
		if err != nil {
			return err
		}
		result.Items = append(result.Items, SyncItem{
			PublicPath: publicPath, Title: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), ContentType: ContentTypePDF,
		})
		return nil
	})
}

func publicPath(categoryRoot, filePath string) (string, error) {
	contentRoot := filepath.Dir(filepath.Dir(categoryRoot))
	relativePath, err := filepath.Rel(contentRoot, filePath)
	if err != nil {
		return "", err
	}
	value := "/" + filepath.ToSlash(relativePath)
	canonical, err := CanonicalizePath(value)
	if err != nil {
		return "", err
	}
	return canonical, nil
}

func resolveArticleLink(basePath, href string) (string, bool) {
	value := strings.TrimSpace(href)
	if value == "" {
		return "", false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "", false
	}
	candidate := parsed.Path
	if !strings.HasPrefix(candidate, "/") {
		candidate = basePath + "/" + candidate
	}
	canonical, err := CanonicalizePath(candidate)
	if err != nil || !IsArticlePath(canonical) {
		return "", false
	}
	return canonical, true
}

type Link struct {
	Href string
	Text string
}

// BookPostLinks reads only anchors nested in the main .book-post element.
// The archived files have regular HTML; this bounded tokenizer intentionally
// avoids accepting links from the duplicated side navigation.
func BookPostLinks(document string) []Link {
	links := make([]Link, 0)
	depth := 0
	bookPostDepth := -1
	var active *Link
	for len(document) > 0 {
		start := strings.IndexByte(document, '<')
		if start < 0 {
			if active != nil {
				active.Text += document
			}
			break
		}
		if active != nil && start > 0 {
			active.Text += document[:start]
		}
		document = document[start:]
		end := tagEnd(document)
		if end < 0 {
			break
		}
		tag := document[1:end]
		document = document[end+1:]
		name, closing, selfClosing, attrs := parseTag(tag)
		if name == "" {
			continue
		}
		if closing {
			if name == "a" && active != nil {
				active.Text = strings.TrimSpace(html.UnescapeString(stripTags(active.Text)))
				links = append(links, *active)
				active = nil
			}
			if !selfClosing && depth > 0 {
				depth--
				if bookPostDepth >= depth {
					bookPostDepth = -1
				}
			}
			continue
		}
		if name == "div" && classContains(attrs["class"], "book-post") && bookPostDepth < 0 {
			bookPostDepth = depth
		}
		if name == "a" && bookPostDepth >= 0 && depth > bookPostDepth && active == nil {
			if href := attrs["href"]; href != "" {
				active = &Link{Href: html.UnescapeString(href)}
			}
		}
		if !selfClosing && !voidTag(name) {
			depth++
		}
	}
	return links
}

func tagEnd(value string) int {
	quote := byte(0)
	for index := 1; index < len(value); index++ {
		if quote != 0 {
			if value[index] == quote {
				quote = 0
			}
			continue
		}
		if value[index] == '\'' || value[index] == '"' {
			quote = value[index]
		} else if value[index] == '>' {
			return index
		}
	}
	return -1
}

func parseTag(value string) (name string, closing bool, selfClosing bool, attrs map[string]string) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "!") || strings.HasPrefix(value, "?") {
		return "", false, false, nil
	}
	closing = strings.HasPrefix(value, "/")
	value = strings.TrimSpace(strings.TrimPrefix(value, "/"))
	selfClosing = strings.HasSuffix(value, "/")
	value = strings.TrimSpace(strings.TrimSuffix(value, "/"))
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", closing, selfClosing, nil
	}
	name = strings.ToLower(fields[0])
	attrs = parseAttributes(strings.TrimSpace(strings.TrimPrefix(value, fields[0])))
	return name, closing, selfClosing, attrs
}

func parseAttributes(value string) map[string]string {
	attrs := make(map[string]string)
	for len(value) > 0 {
		value = strings.TrimLeftFunc(value, unicode.IsSpace)
		if value == "" {
			break
		}
		keyEnd := strings.IndexFunc(value, func(r rune) bool { return unicode.IsSpace(r) || r == '=' })
		if keyEnd < 0 {
			break
		}
		key := strings.ToLower(value[:keyEnd])
		value = strings.TrimLeftFunc(value[keyEnd:], unicode.IsSpace)
		if !strings.HasPrefix(value, "=") {
			continue
		}
		value = strings.TrimLeftFunc(value[1:], unicode.IsSpace)
		if value == "" {
			break
		}
		if value[0] == '\'' || value[0] == '"' {
			quote := value[0]
			value = value[1:]
			end := strings.IndexByte(value, quote)
			if end < 0 {
				break
			}
			attrs[key] = value[:end]
			value = value[end+1:]
			continue
		}
		end := strings.IndexFunc(value, unicode.IsSpace)
		if end < 0 {
			attrs[key] = value
			break
		}
		attrs[key] = value[:end]
		value = value[end:]
	}
	return attrs
}

func classContains(value string, want string) bool {
	for _, class := range strings.Fields(value) {
		if class == want {
			return true
		}
	}
	return false
}

func voidTag(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

func stripTags(value string) string {
	for {
		start := strings.IndexByte(value, '<')
		if start < 0 {
			return value
		}
		end := strings.IndexByte(value[start:], '>')
		if end < 0 {
			return value[:start]
		}
		value = value[:start] + value[start+end+1:]
	}
}
