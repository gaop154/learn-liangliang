package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBookPostLinksExcludesNavigation(t *testing.T) {
	document := `
<div class="book-menu"><a href="/专栏/课程/导航.md.html">导航</a></div>
<div class="book-post"><ul>
<li><a href="/专栏/课程/00 开篇词.md.html">00 开篇词</a></li>
<li><a href="/专栏/课程/01 正文.md.html">01 <strong>正文</strong></a></li>
</ul></div>
`
	links := BookPostLinks(document)
	if len(links) != 2 {
		t.Fatalf("正文链接数 = %d, want 2", len(links))
	}
	if links[0].Href != "/专栏/课程/00 开篇词.md.html" || links[0].Text != "00 开篇词" {
		t.Fatalf("第一条正文链接 = %#v", links[0])
	}
	if links[1].Text != "01 正文" {
		t.Fatalf("嵌套标签文本解析失败: %#v", links[1])
	}
}

func TestBuildSyncRequiresNewContentLayout(t *testing.T) {
	if _, err := BuildSync(t.TempDir()); err == nil {
		t.Fatal("缺少新内容分类目录时应拒绝同步")
	}
}

func TestBuildSyncUsesNewContentLayoutAndBookPostOrder(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "其他"), 0o755); err != nil {
		t.Fatalf("创建其他目录失败: %v", err)
	}
	mustWriteFile(t, filepath.Join(root, "专栏", "课程 A", "index.html"), `
<div class="book-menu"><a href="/专栏/课程 A/99 导航.md.html">导航</a></div>
<div class="book-post"><a href="/专栏/课程 A/01 正文.md.html">第一篇</a><a href="/专栏/课程 A/02 正文.md.html">第二篇</a></div>`)
	mustWriteFile(t, filepath.Join(root, "专栏", "课程 A", "01 正文.md.html"), "文章")
	mustWriteFile(t, filepath.Join(root, "专栏", "课程 A", "02 正文.md.html"), "文章")
	mustWriteFile(t, filepath.Join(root, "其他", "文章", "index.html"), `<div class="book-post"><a href="/其他/文章/普通.md.html">普通文章</a></div>`)
	mustWriteFile(t, filepath.Join(root, "其他", "文章", "普通.md.html"), "文章")
	mustWriteFile(t, filepath.Join(root, "其他", "PDF", "资料.pdf"), "pdf")

	source, err := BuildSync(root)
	if err != nil {
		t.Fatalf("BuildSync() 返回错误: %v", err)
	}
	if len(source.Courses) != 1 || source.Courses[0].PublicPath != "/专栏/课程 A" {
		t.Fatalf("课程索引错误: %#v", source.Courses)
	}
	if len(source.CourseArticles) != 2 || source.CourseArticles[0].ArticlePath != "/专栏/课程 A/01 正文.md.html" || source.CourseArticles[1].Position != 1 {
		t.Fatalf("课程章节顺序错误: %#v", source.CourseArticles)
	}
	if len(source.Items) != 4 {
		t.Fatalf("内容索引数量 = %d, want 4", len(source.Items))
	}
	if source.Items[2].ContentType != ContentTypeArticle || source.Items[3].ContentType != ContentTypePDF {
		t.Fatalf("其他内容类型错误: %#v", source.Items)
	}
}

func mustWriteFile(t *testing.T, filePath string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
}

func TestBuildSyncUsesOtherCategoryIndexAsArticleSource(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "专栏", "课程", "index.html"), `<div class="book-post"></div>`)
	mustWriteFile(t, filepath.Join(root, "其他", "文章", "index.html"), `<div class="book-post"><a href="/其他/文章/已列出.md.html">已列出</a></div>`)
	mustWriteFile(t, filepath.Join(root, "其他", "文章", "已列出.md.html"), "文章")
	mustWriteFile(t, filepath.Join(root, "其他", "文章", "未列出.md.html"), "文章")

	source, err := BuildSync(root)
	if err != nil {
		t.Fatalf("BuildSync() 返回错误: %v", err)
	}
	if len(source.Items) != 1 || source.Items[0].PublicPath != "/其他/文章/已列出.md.html" {
		t.Fatalf("其他分类必须仅按索引正文链接建档: %#v", source.Items)
	}
}

func TestResolveArticleLinkRejectsOutsideCourseAndPDF(t *testing.T) {
	for _, href := range []string{"/其他/PDF/资料.pdf", "https://example.com/A.md.html", "../其他/文章/A.md.html"} {
		if _, ok := resolveArticleLink("/专栏/课程", href); ok {
			t.Fatalf("%q 不应解析为课程文章", href)
		}
	}
	got, ok := resolveArticleLink("/专栏/课程", "01 文章.md.html")
	if !ok || got != "/专栏/课程/01 文章.md.html" {
		t.Fatalf("相对课程文章 = %q, %t", got, ok)
	}
}
