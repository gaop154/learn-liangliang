package catalog

import "testing"

func TestArticlePathEligibilityUsesNewPublicPaths(t *testing.T) {
	valid := []string{
		"/专栏/课程/01 文章.md.html",
		"/其他/文章/文章.md.html",
		"/其他/极客时间/文章.md.html",
		"/其他/恋爱必修课/文章.md.html",
	}
	for _, value := range valid {
		if !IsArticlePath(value) {
			t.Fatalf("%q 应为合法文章路径", value)
		}
	}
	invalid := []string{
		"/专栏/课程/index.html",
		"/专栏/课程/目录/文章.md.html",
		"/其他/PDF/资料.pdf",
		"/其他/PDF/伪造.md.html",
		"/文章/旧路径.md.html",
		"/极客时间/旧路径.md.html",
		"/恋爱必修课/旧路径.md.html",
	}
	for _, value := range invalid {
		if IsArticlePath(value) {
			t.Fatalf("%q 不应为合法文章路径", value)
		}
	}
}

func TestCanonicalizePathPreservesCanonicalContentCompatibility(t *testing.T) {
	got, err := CanonicalizePath("/content/%E4%B8%93%E6%A0%8F/%E8%AF%BE%E7%A8%8B/01%20%E6%96%87%E7%AB%A0.md.html")
	if err != nil {
		t.Fatalf("规范化路径失败: %v", err)
	}
	if want := "/专栏/课程/01 文章.md.html"; got != want {
		t.Fatalf("路径 = %q, want %q", got, want)
	}
	for _, value := range []string{"/content/../其他/文章/A.md.html", "/其他/%00/文章.md.html", "/其他/%ZZ/文章.md.html"} {
		if _, err := CanonicalizePath(value); err == nil {
			t.Fatalf("%q 应被拒绝", value)
		}
	}
}
