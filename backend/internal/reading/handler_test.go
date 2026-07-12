package reading

import "testing"

func TestCanonicalizeArticlePath(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "content 物理路径归一为公开路径",
			raw:  "/content/%E4%B8%93%E6%A0%8F/%E8%AF%BE%E7%A8%8B/01%20%E7%A4%BA%E4%BE%8B.md.html",
			want: "/专栏/课程/01 示例.md.html",
		},
		{
			name: "反斜杠归一化",
			raw:  "content\\专栏\\课程\\01 示例.md.html",
			want: "/专栏/课程/01 示例.md.html",
		},
		{
			name: "重复斜杠与当前目录归一化",
			raw:  "/content//专栏/课程/./01%20示例.md.html",
			want: "/专栏/课程/01 示例.md.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonicalizeArticlePath(tt.raw)
			if err != nil {
				t.Fatalf("canonicalizeArticlePath() 返回错误: %v", err)
			}
			if got != tt.want {
				t.Fatalf("canonicalizeArticlePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestArticleAndCoursePathEligibility(t *testing.T) {
	for _, articlePath := range []string{
		"/专栏/课程/01 示例.md.html",
		"/其他/文章/Java 示例.md.html",
		"/其他/极客时间/课程.md.html",
		"/其他/恋爱必修课/01 示例.md.html",
	} {
		if !isArticlePath(articlePath) {
			t.Fatalf("文章路径 %q 应被识别为文章", articlePath)
		}
	}
	for _, articlePath := range []string{
		"/专栏",
		"/专栏/课程",
		"/其他/文章/index.html",
		"/文章/旧路径.md.html",
		"/极客时间/旧路径.md.html",
		"/恋爱必修课/旧路径.md.html",
		"/其他/PDF/资料.pdf",
		"/其他/PDF/伪造.md.html",
		"/static/伪造.md.html",
		"/assets/伪造.md.html",
	} {
		if isArticlePath(articlePath) {
			t.Fatalf("非文章路径 %q 不应被识别为文章", articlePath)
		}
	}

	if !isCoursePath("/专栏/课程 名称") {
		t.Fatal("单段专栏课程路径应合法")
	}
	for _, coursePath := range []string{"/专栏", "/专栏/", "/专栏/课程/章节.md.html", "/文章/课程", "/专栏/课程/额外层级"} {
		if isCoursePath(coursePath) {
			t.Fatalf("非法课程路径 %q 不应通过校验", coursePath)
		}
	}
}

func TestCanonicalizeArticlePathRejectsMalformedAndUnsafePaths(t *testing.T) {
	for _, rawPath := range []string{
		"/专栏/%ZZ/01.md.html",
		"/专栏/%00/01.md.html",
		"/专栏/课程/../01.md.html",
		"/content/../static/伪造.md.html",
	} {
		if _, err := canonicalizeArticlePath(rawPath); err == nil {
			t.Fatalf("非法路径 %q 应被拒绝", rawPath)
		}
	}
}

func TestBatchPathValidation(t *testing.T) {
	paths, err := canonicalizeArticlePaths([]string{
		"/content/专栏/课程/01 示例.md.html",
		"/专栏/课程/02 示例.md.html",
	})
	if err != nil {
		t.Fatalf("合法文章集合不应返回错误: %v", err)
	}
	if paths[0] != "/专栏/课程/01 示例.md.html" {
		t.Fatalf("文章路径未正确归一化: %q", paths[0])
	}

	tooManyPaths := make([]string, maxBatchPaths+1)
	for _, paths := range [][]string{
		nil,
		{"/专栏/课程"},
		{"/专栏/课程/01.md.html", "/content/专栏/课程/01.md.html"},
		tooManyPaths,
	} {
		if _, err := canonicalizeArticlePaths(paths); err == nil {
			t.Fatalf("非法文章集合 %v 应被拒绝", paths)
		}
	}
}

func TestCoursePathValidation(t *testing.T) {
	paths, err := canonicalizeCoursePaths([]string{
		"/content/专栏/课程 A",
		"/专栏/课程 B",
	})
	if err != nil {
		t.Fatalf("合法课程集合不应返回错误: %v", err)
	}
	if paths[0] != "/专栏/课程 A" {
		t.Fatalf("课程路径未正确归一化: %q", paths[0])
	}

	for _, paths := range [][]string{
		nil,
		{"/专栏"},
		{"/专栏/课程/01.md.html"},
		{"/专栏/课程", "/content/专栏/课程"},
	} {
		if _, err := canonicalizeCoursePaths(paths); err == nil {
			t.Fatalf("非法课程集合 %v 应被拒绝", paths)
		}
	}
}

func TestFinishedRequiresOneHundredPercent(t *testing.T) {
	valid := upsertRequest{
		ArticlePath:     "/专栏/课程/01 示例.md.html",
		ArticleTitle:    "01 示例",
		ProgressPercent: 100,
		Finished:        true,
	}
	if err := validateRequest(valid); err != nil {
		t.Fatalf("100%% 完成状态应合法: %v", err)
	}

	valid.ProgressPercent = 99
	if err := validateRequest(valid); err == nil {
		t.Fatal("99% 不应允许写入 finished=true")
	}
}
