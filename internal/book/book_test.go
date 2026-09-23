package book

import (
	"regexp"
	"strings"
	"testing"

	"ezpub/internal/lang"
)

func TestSplitChaptersByRegex(t *testing.T) {
	text := "序言\n這是開頭\n第一章 開始\n第一段\n第二章 繼續\n第二段"
	bk, err := FromText(text, TextOptions{
		Title:           "測試書",
		Language:        "zh-TW",
		ChapterRegex:    `^\s*第[一二三四五六七八九十]+章.*`,
		RemoveBlankLine: true,
		Layout:          LayoutOptions{LineHeightPercent: 130, FontSizePercent: 100, IndentEm: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Chapters) != 3 {
		t.Fatalf("chapter count = %d, want 3", len(bk.Chapters))
	}
	if bk.Chapters[0].Title != lang.PrefaceTitle {
		t.Fatalf("preface title = %q, want %s", bk.Chapters[0].Title, lang.PrefaceTitle)
	}
	if bk.Chapters[1].Title != "第一章 開始" {
		t.Fatalf("first chapter = %q", bk.Chapters[1].Title)
	}
}

func TestRawHTMLTag(t *testing.T) {
	blocks := paragraphs("##<img src=\"../images/a.jpg\" alt=\"a\"/>", TextOptions{
		EnableRawHTML: true,
		RawHTMLTag:    "##",
	})
	if len(blocks) != 1 || !blocks[0].RawHTML {
		t.Fatalf("raw block not detected: %#v", blocks)
	}
}

func TestFromTextDropsInvalidXMLControlChars(t *testing.T) {
	bk, err := FromText("第一章\x00\x0f\n內容\x01", TextOptions{Title: "測試書", ChapterRegex: `^第一章.*`})
	if err != nil {
		t.Fatal(err)
	}
	if got := bk.Chapters[0].Title; got != "第一章" {
		t.Fatalf("chapter title = %q", got)
	}
	if got := bk.Chapters[0].Blocks[0].Text; got != "內容" {
		t.Fatalf("chapter text = %q", got)
	}
}

func TestAddFullWidthSpaces(t *testing.T) {
	blocks := paragraphs("內容", TextOptions{AddSpace: true, AddSpaceCount: 2})
	if len(blocks) != 1 || blocks[0].Text != "　　內容" {
		t.Fatalf("block = %#v, want two full-width spaces", blocks)
	}
}

func TestExternalFontSources(t *testing.T) {
	bk, err := FromText("內容", TextOptions{
		Title:       "測試書",
		FontSources: []string{"res:///fonts/a.ttf", "res:///fonts/b.ttf"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`url("res:///fonts/a.ttf")`,
		`url("res:///fonts/b.ttf")`,
		`font-family: "ezpub-reader-font"`,
	} {
		if !strings.Contains(bk.CSS, want) {
			t.Fatalf("css missing %q:\n%s", want, bk.CSS)
		}
	}
}

func TestExternalFontSourcesDoNotDuplicateBodySelector(t *testing.T) {
	bk, err := FromText("內容", TextOptions{
		Title:       "測試書",
		FontSources: []string{"res:///system/fonts/DroidSansFallback.ttf"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if count := regexp.MustCompile(`(?m)^body\s*\{`).FindAllStringIndex(bk.CSS, -1); len(count) != 1 {
		t.Fatalf("body selector count = %d, want 1:\n%s", len(count), bk.CSS)
	}
	if !strings.Contains(bk.CSS, `font-family: "ezpub-reader-font", serif;`) {
		t.Fatalf("css missing merged reader font family:\n%s", bk.CSS)
	}
}

func TestCustomCSSModes(t *testing.T) {
	skipped, err := FromText("內容", TextOptions{
		Title:     "測試書",
		CustomCSS: "body { color: red; }",
		CSSMode:   0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(skipped.CSS, "color: red") {
		t.Fatalf("css mode 0 should skip custom css:\n%s", skipped.CSS)
	}

	appended, err := FromText("內容", TextOptions{
		Title:     "測試書",
		CustomCSS: "body { color: red; }",
		CSSMode:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(appended.CSS, "color: red") {
		t.Fatalf("css mode 1 should append custom css:\n%s", appended.CSS)
	}
}

func TestLayoutCSSUsesConfiguredUnits(t *testing.T) {
	css := DefaultCSS(LayoutOptions{
		LineHeightPercent: 130,
		FontSizePercent:   100,
		IndentEm:          2,
		MarginTop:         CSSLength{Value: 10, Unit: "px"},
		MarginRight:       CSSLength{Value: 1, Unit: "em"},
		MarginBottom:      CSSLength{Value: 2, Unit: "%"},
		MarginLeft:        CSSLength{Value: 0, Unit: "px"},
		ParagraphSpacing:  CSSLength{Value: 1.5, Unit: "em"},
	})
	for _, want := range []string{
		"margin-top: 10px;",
		"margin-bottom: 2%;",
		"margin-left: 0px;",
		"margin-right: 1em;",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("css missing configured margin %q:\n%s", want, css)
		}
	}
	if strings.Contains(css, "margin: 1.5em 0 0;") {
		t.Fatalf("css should no longer use shorthand paragraph spacing:\n%s", css)
	}
	if !strings.Contains(css, "margin-top: 1.5em;") {
		t.Fatalf("css missing paragraph spacing:\n%s", css)
	}
}

func TestDefaultCSSIncludesTOCLevelStyles(t *testing.T) {
	css := DefaultCSS(LayoutOptions{})
	for _, want := range []string{
		".titletoc",
		".titlel2std",
		".toc {",
		".toc ol {",
		".tocl1",
		".tocl2",
		".tocl3",
		".tocl4",
		"border-color: #939E92;",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("css missing %q:\n%s", want, css)
		}
	}
}

func TestSplitChaptersByPlan(t *testing.T) {
	bk, err := FromText("序言\n第一章 開始\n內容\n第二章 繼續\n後續", TextOptions{
		Title: "測試書",
		ChapterPlan: []ChapterMark{
			{StartLine: 1, Level: 1, Title: "序"},
			{StartLine: 2, Level: 1, Title: "手動一", SkipHeading: true},
			{StartLine: 4, Level: 1, Title: "手動二", SkipHeading: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Chapters) != 3 {
		t.Fatalf("chapter count = %d, want 3", len(bk.Chapters))
	}
	if got := bk.Chapters[1].Blocks[0].Text; got != "內容" {
		t.Fatalf("chapter body = %q, want 內容", got)
	}
}

func TestSplitChaptersFixed(t *testing.T) {
	bk, err := FromText("12345\n67890\nabc", TextOptions{
		Title:           "測試書",
		FixedChapterLen: 6,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Chapters) < 2 {
		t.Fatalf("chapter count = %d, want at least 2", len(bk.Chapters))
	}
}

func TestSplitChaptersEven(t *testing.T) {
	bk, err := FromText("11111\n22222\n33333\n44444", TextOptions{
		Title:      "測試書",
		SplitCount: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Chapters) != 2 {
		t.Fatalf("chapter count = %d, want 2", len(bk.Chapters))
	}
}

func TestSkipEmptyChapters(t *testing.T) {
	bk, err := FromText("第一章 空\n第二章 有\n內容", TextOptions{
		Title:             "測試書",
		ChapterRegex:      `^第[一二]章.*`,
		RemoveBlankLine:   true,
		SkipEmptyChapters: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Chapters) != 1 {
		t.Fatalf("chapter count = %d, want 1", len(bk.Chapters))
	}
	if bk.Chapters[0].Title != "第二章 有" {
		t.Fatalf("chapter title = %q, want 第二章 有", bk.Chapters[0].Title)
	}
}

func TestEmptyChaptersAsSubdirectory(t *testing.T) {
	bk, err := FromText("第一章 空\n第二章 有\n內容", TextOptions{
		Title:               "測試書",
		ChapterRegex:        `^第[一二]章.*`,
		RemoveBlankLine:     true,
		EmptyAsSubdirectory: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Chapters) != 2 {
		t.Fatalf("chapter count = %d, want 2", len(bk.Chapters))
	}
	if !bk.Chapters[0].TOCOnly {
		t.Fatalf("empty chapter should be TOC-only: %#v", bk.Chapters[0])
	}
	if bk.Chapters[1].TOCOnly {
		t.Fatalf("non-empty chapter should be readable: %#v", bk.Chapters[1])
	}
}

func TestRewriteImageRefs(t *testing.T) {
	bk, err := FromText("##<p><img src=\"art/one.jpg\" alt=\"one\"/></p>", TextOptions{
		Title:         "測試書",
		EnableRawHTML: true,
		RawHTMLTag:    "##",
		ImageResolver: map[string]string{
			"art/one.jpg": `C:\book\art\one.jpg`,
			"one.jpg":     `C:\book\art\one.jpg`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Images) != 1 {
		t.Fatalf("image count = %d, want 1", len(bk.Images))
	}
	if got := bk.Chapters[0].Blocks[0].Text; got != `<p><img src="../images/one.jpg" alt="one"/></p>` {
		t.Fatalf("rewritten html = %q", got)
	}
}
