package epub

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ezpub/internal/book"
)

func TestWriteAndExtractText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk, err := book.FromText("第一章 開始\n內容", book.TextOptions{
		Title:           "測試書",
		Translator:      "譯者",
		Description:     "簡介",
		Publisher:       "出版社",
		Subjects:        []string{"小說"},
		Rights:          "版權",
		Language:        "zh-TW",
		ChapterRegex:    `^第一章.*`,
		RemoveBlankLine: true,
		Layout:          book.LayoutOptions{LineHeightPercent: 130, FontSizePercent: 100, IndentEm: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	assertZipEntry(t, path, "mimetype")
	assertZipEntry(t, path, "META-INF/container.xml")
	assertZipEntry(t, path, "EPUB/content.opf")
	assertZipEntry(t, path, "EPUB/text/nav.xhtml")
	assertZipEntry(t, path, "EPUB/text/book-toc.xhtml")
	assertZipEntry(t, path, "EPUB/toc.ncx")
	opf := readZipEntry(t, path, "EPUB/content.opf")
	for _, want := range []string{
		`<item id="htmltoc" href="text/book-toc.xhtml" media-type="application/xhtml+xml"/>`,
		`<itemref idref="htmltoc"/>`,
		`<reference type="toc" title="Table Of Contents" href="text/book-toc.xhtml"/>`,
		"<dc:publisher>出版社</dc:publisher>",
		"<dc:contributor>譯者</dc:contributor>",
		"<dc:subject>小說</dc:subject>",
		"<dc:description>簡介</dc:description>",
		"<dc:rights>版權</dc:rights>",
	} {
		if !strings.Contains(opf, want) {
			t.Fatalf("content.opf missing %q:\n%s", want, opf)
		}
	}
	chapter := readZipEntry(t, path, "EPUB/text/chapter0001.xhtml")
	for _, want := range []string{
		`<h2 id="title" class="titlel2std">第一章 開始</h2>`,
		`<p class="a">內容</p>`,
	} {
		if !strings.Contains(chapter, want) {
			t.Fatalf("chapter xhtml missing %q:\n%s", want, chapter)
		}
	}
	booktoc := readZipEntry(t, path, "EPUB/text/book-toc.xhtml")
	for _, want := range []string{
		`<h2 class="titletoc">目錄</h2>`,
		`<div class="toc">`,
		`<dt class="tocl2"><a href="chapter0001.xhtml">第一章 開始</a></dt>`,
		`<dd></dd>`,
	} {
		if !strings.Contains(booktoc, want) {
			t.Fatalf("book-toc.xhtml missing %q:\n%s", want, booktoc)
		}
	}

	text, err := ExtractText(path)
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("extracted text is empty")
	}
}

func TestExtractTextUsesOPFSpineForHTMLFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spine-html.epub")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for name, data := range map[string]string{
		"META-INF/container.xml": `<?xml version="1.0" encoding="utf-8"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="second" href="Text/chapter_2.html" media-type="application/xhtml+xml"/>
    <item id="first" href="Text/chapter_1.html" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="second"/><itemref idref="first"/></spine>
</package>`,
		"OEBPS/nav.xhtml": `<html xmlns="http://www.w3.org/1999/xhtml"><body><nav><ol><li>目錄</li></ol></nav></body></html>`,
		"OEBPS/Text/chapter_1.html": `<html xmlns="http://www.w3.org/1999/xhtml"><body><p>第一個檔名</p></body></html>`,
		"OEBPS/Text/chapter_2.html": `<html xmlns="http://www.w3.org/1999/xhtml"><body><p>第二個檔名但 spine 第一</p></body></html>`,
	} {
		if err := addText(zw, name, data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	text, err := ExtractText(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "第二個檔名但 spine 第一") || !strings.Contains(text, "第一個檔名") {
		t.Fatalf("missing extracted body text:\n%s", text)
	}
	if strings.Index(text, "第二個檔名但 spine 第一") > strings.Index(text, "第一個檔名") {
		t.Fatalf("extract order should follow spine, not filename:\n%s", text)
	}
	if strings.Contains(text, "目錄") {
		t.Fatalf("nav text should not be extracted:\n%s", text)
	}
}

func TestWriteNCXUIDMatchesExplicitIdentifier(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk, err := book.FromText("第一章 開始\n內容", book.TextOptions{
		Title:        "測試書",
		Identifier:   "urn:uuid:593774c3-91c0-4dad-9071-35d0456655af",
		ChapterRegex: `^第一章.*`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	opf := readZipEntry(t, path, "EPUB/content.opf")
	ncx := readZipEntry(t, path, "EPUB/toc.ncx")
	want := `urn:uuid:593774c3-91c0-4dad-9071-35d0456655af`
	if !strings.Contains(opf, `<dc:identifier id="bookid">`+want+`</dc:identifier>`) {
		t.Fatalf("content.opf should use explicit identifier:\n%s", opf)
	}
	if !strings.Contains(ncx, `<meta name="dtb:uid" content="`+want+`"/>`) {
		t.Fatalf("toc.ncx should use the OPF book identifier:\n%s", ncx)
	}
}

func TestWriteCanSkipTextCoverPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk, err := book.FromText("第一章 開始\n內容", book.TextOptions{
		Title:         "測試書",
		ChapterRegex:  `^第一章.*`,
		SkipCoverPage: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	assertZipEntryMissing(t, path, "EPUB/text/cover.xhtml")
	opf := readZipEntry(t, path, "EPUB/content.opf")
	if strings.Contains(opf, "cover-page") {
		t.Fatalf("content.opf should not reference cover-page:\n%s", opf)
	}
}

func TestWriteImageCoverMetadata(t *testing.T) {
	dir := t.TempDir()
	cover := filepath.Join(dir, "cover.jpg")
	if err := os.WriteFile(cover, []byte("jpeg"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "book.epub")
	bk, err := book.FromText("第一章 開始\n內容", book.TextOptions{
		Title:        "測試書",
		Author:       "作者",
		ChapterRegex: `^第一章.*`,
		CoverPath:    cover,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	assertZipEntry(t, path, "EPUB/images/cover.jpg")
	assertZipEntry(t, path, "EPUB/text/cover.xhtml")
	opf := readZipEntry(t, path, "EPUB/content.opf")
	for _, want := range []string{
		`<meta name="cover" content="cover-image"/>`,
		`<itemref idref="cover-page"/>`,
		`<reference type="cover" title="Cover" href="text/cover.xhtml"/>`,
	} {
		if !strings.Contains(opf, want) {
			t.Fatalf("content.opf missing %q:\n%s", want, opf)
		}
	}
	ncx := readZipEntry(t, path, "EPUB/toc.ncx")
	for _, want := range []string{
		`<meta name="cover" content="cover-page"/>`,
		`<docAuthor><text>作者</text></docAuthor>`,
		`<navPoint id="cover" playOrder="1">`,
		`<content src="text/cover.xhtml"/>`,
		`<navPoint id="htmltoc" playOrder="2">`,
		`<content src="text/book-toc.xhtml"/>`,
	} {
		if !strings.Contains(ncx, want) {
			t.Fatalf("toc.ncx missing %q:\n%s", want, ncx)
		}
	}
	coverXHTML := readZipEntry(t, path, "EPUB/text/cover.xhtml")
	if !strings.Contains(coverXHTML, `class="wedge"`) || !strings.Contains(coverXHTML, `<table style="height:720px`) {
		t.Fatalf("cover.xhtml should center image with table:\n%s", coverXHTML)
	}
}

func TestWriteMultipleCreators(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk, err := book.FromText("第一章\n內容", book.TextOptions{
		Title:        "測試書",
		Author:       " 作者一, 作者二 ， 作者一 ",
		ChapterRegex: `^第一章.*`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := bk.Author, "作者一, 作者二"; got != want {
		t.Fatalf("author = %q, want %q", got, want)
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	opf := readZipEntry(t, path, "EPUB/content.opf")
	for _, want := range []string{
		`<dc:creator>作者一</dc:creator>`,
		`<dc:creator>作者二</dc:creator>`,
	} {
		if !strings.Contains(opf, want) {
			t.Fatalf("content.opf missing %q:\n%s", want, opf)
		}
	}
}

func TestWriteTextCoverMatchesEasyPubStructure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk, err := book.FromText("第一章 開始\n內容", book.TextOptions{
		Title:        "巴黎茶花女遺事",
		Author:       "小仲馬",
		ChapterRegex: `^第一章.*`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	cover := readZipEntry(t, path, "EPUB/text/cover.xhtml")
	for _, want := range []string{
		`<h1 class="booktitle">巴黎茶花女遺事</h1>`,
		`<h3 class="bookauthor">小仲馬</h3>`,
	} {
		if !strings.Contains(cover, want) {
			t.Fatalf("cover.xhtml missing %q:\n%s", want, cover)
		}
	}
	css := readZipEntry(t, path, "EPUB/styles/style.css")
	for _, want := range []string{".booktitle", ".bookauthor"} {
		if !strings.Contains(css, want) {
			t.Fatalf("style.css missing %q:\n%s", want, css)
		}
	}
}

func TestWriteFixtureCoverImage(t *testing.T) {
	cover := filepath.Join("..", "..", "test_file", "cover.jpg")
	coverBytes, err := os.ReadFile(cover)
	if err != nil {
		t.Skipf("fixture cover not available: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk, err := book.FromText("第一章 開始\n內容", book.TextOptions{
		Title:        "巴黎茶花女遺事",
		Author:       "小仲馬",
		ChapterRegex: `^第一章.*`,
		CoverPath:    cover,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	got := readZipEntryBytes(t, path, "EPUB/images/cover.jpg")
	if !bytes.Equal(got, coverBytes) {
		t.Fatal("cover image bytes in epub differ from fixture")
	}
}

func TestWriteSplitsLargeFlows(t *testing.T) {
	ch := book.Chapter{
		ID:    "chapter0001",
		Title: "第一章",
		Blocks: []book.Block{
			{Text: strings.Repeat("一", 40)},
			{Text: strings.Repeat("二", 40)},
			{Text: strings.Repeat("三", 40)},
		},
	}
	parts := splitChapterBySize(ch, 90)
	if len(parts) < 2 {
		t.Fatalf("part count = %d, want at least 2", len(parts))
	}
	bk := &book.Book{Chapters: []book.Chapter{ch}, FlowSizeKB: 1}
	docs := chapterDocs(bk)
	if docs[0].ID != "chapter0001" {
		t.Fatalf("first doc id = %q", docs[0].ID)
	}
}

func TestWriteTOCOnlyChapter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk := &book.Book{
		Title: "測試書",
		Chapters: []book.Chapter{
			{ID: "chapter0001", Title: "第一章 空", Level: 1, TOCOnly: true},
			{ID: "chapter0002", Title: "第二章 有", Level: 1, Blocks: []book.Block{{Text: "內容"}}},
		},
		CSS: book.DefaultCSS(book.LayoutOptions{}),
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	assertZipEntryMissing(t, path, "EPUB/text/chapter0001.xhtml")
	assertZipEntry(t, path, "EPUB/text/chapter0002.xhtml")
	nav := readZipEntry(t, path, "EPUB/text/nav.xhtml")
	if !strings.Contains(nav, `<span>第一章 空</span>`) {
		t.Fatalf("nav should render TOC-only chapter as span:\n%s", nav)
	}
	ncx := readZipEntry(t, path, "EPUB/toc.ncx")
	if !strings.Contains(ncx, `<content src="text/chapter0002.xhtml"/>`) {
		t.Fatalf("ncx should point TOC-only chapter to next readable chapter:\n%s", ncx)
	}
	opf := readZipEntry(t, path, "EPUB/content.opf")
	if strings.Contains(opf, `id="chapter0001"`) {
		t.Fatalf("opf should not include TOC-only chapter manifest item:\n%s", opf)
	}
}

func TestWriteNestedTOCFromChapterLevels(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk := &book.Book{
		Title: "測試書",
		Chapters: []book.Chapter{
			{ID: "chapter0001", Title: "第一節", Level: 1, Blocks: []book.Block{{Text: "一"}}},
			{ID: "chapter0002", Title: "第二節", Level: 2, Blocks: []book.Block{{Text: "二"}}},
			{ID: "chapter0003", Title: "第三節", Level: 2, Blocks: []book.Block{{Text: "三"}}},
			{ID: "chapter0004", Title: "第四節", Level: 1, Blocks: []book.Block{{Text: "四"}}},
		},
		CSS: book.DefaultCSS(book.LayoutOptions{}),
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	nav := readZipEntry(t, path, "EPUB/text/nav.xhtml")
	for _, want := range []string{
		`<li class="tocl2"><a href="chapter0001.xhtml">第一節</a>`,
		`<li class="tocl3"><a href="chapter0002.xhtml">第二節</a></li>`,
		`<li class="tocl3"><a href="chapter0003.xhtml">第三節</a></li>`,
		`</ol>`,
	} {
		if !strings.Contains(nav, want) {
			t.Fatalf("nav.xhtml missing %q:\n%s", want, nav)
		}
	}
	first := strings.Index(nav, `<li class="tocl2"><a href="chapter0001.xhtml">第一節</a>`)
	child := strings.Index(nav, `<li class="tocl3"><a href="chapter0002.xhtml">第二節</a></li>`)
	closeParent := strings.Index(nav, `<li class="tocl2"><a href="chapter0004.xhtml">第四節</a>`)
	if first < 0 || child < first || closeParent < child {
		t.Fatalf("nav.xhtml should keep level 2 items under the first level 1 item:\n%s", nav)
	}

	ncx := readZipEntry(t, path, "EPUB/toc.ncx")
	for _, want := range []string{
		`<meta name="dtb:depth" content="2"/>`,
		`<navPoint id="htmltoc" playOrder="1">`,
		`<navPoint id="chapter0001" playOrder="2">`,
		`<navLabel><text>　第二節</text></navLabel>`,
		`<navLabel><text>　第三節</text></navLabel>`,
		`</navPoint>`,
	} {
		if !strings.Contains(ncx, want) {
			t.Fatalf("toc.ncx missing %q:\n%s", want, ncx)
		}
	}
	parentStart := strings.Index(ncx, `<navPoint id="chapter0001" playOrder="2">`)
	childStart := strings.Index(ncx, `<navPoint id="chapter0002" playOrder="3">`)
	parentClose := strings.Index(ncx, `<navPoint id="chapter0004" playOrder="5">`)
	if parentStart < 0 || childStart < parentStart || parentClose < childStart {
		t.Fatalf("toc.ncx should nest chapter0002 before the next level 1 navPoint:\n%s", ncx)
	}
}

func TestWriteDeepNestedTOCFromChapterLevels(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.epub")
	bk := &book.Book{
		Title: "測試書",
		Chapters: []book.Chapter{
			{ID: "chapter0001", Title: "第一層", Level: 1, Blocks: []book.Block{{Text: "一"}}},
			{ID: "chapter0002", Title: "第二層", Level: 2, Blocks: []book.Block{{Text: "二"}}},
			{ID: "chapter0003", Title: "第三層", Level: 3, Blocks: []book.Block{{Text: "三"}}},
			{ID: "chapter0004", Title: "第四層", Level: 4, Blocks: []book.Block{{Text: "四"}}},
			{ID: "chapter0005", Title: "回到第一層", Level: 1, Blocks: []book.Block{{Text: "五"}}},
		},
		CSS: book.DefaultCSS(book.LayoutOptions{}),
	}
	if err := Write(path, bk); err != nil {
		t.Fatal(err)
	}
	nav := readZipEntry(t, path, "EPUB/text/nav.xhtml")
	for _, want := range []string{
		`<li class="tocl3"><a href="chapter0002.xhtml">第二層</a>`,
		`<li class="tocl4"><a href="chapter0003.xhtml">第三層</a>`,
		`<li class="tocl4"><a href="chapter0004.xhtml">第四層</a></li>`,
		`<li class="tocl2"><a href="chapter0005.xhtml">回到第一層</a></li>`,
	} {
		if !strings.Contains(nav, want) {
			t.Fatalf("nav.xhtml missing %q:\n%s", want, nav)
		}
	}
	l2 := strings.Index(nav, `<li class="tocl3"><a href="chapter0002.xhtml">第二層</a>`)
	l3 := strings.Index(nav, `<li class="tocl4"><a href="chapter0003.xhtml">第三層</a>`)
	l4 := strings.Index(nav, `<li class="tocl4"><a href="chapter0004.xhtml">第四層</a></li>`)
	nextL1 := strings.Index(nav, `<li class="tocl2"><a href="chapter0005.xhtml">回到第一層</a></li>`)
	if l2 < 0 || l3 < l2 || l4 < l3 || nextL1 < l4 {
		t.Fatalf("nav.xhtml should preserve deep chapter order:\n%s", nav)
	}

	ncx := readZipEntry(t, path, "EPUB/toc.ncx")
	for _, want := range []string{
		`<meta name="dtb:depth" content="4"/>`,
		`<navLabel><text>　第二層</text></navLabel>`,
		`<navLabel><text>　　第三層</text></navLabel>`,
		`<navLabel><text>　　　第四層</text></navLabel>`,
	} {
		if !strings.Contains(ncx, want) {
			t.Fatalf("toc.ncx missing %q:\n%s", want, ncx)
		}
	}
	l2 = strings.Index(ncx, `<navPoint id="chapter0002" playOrder="3">`)
	l3 = strings.Index(ncx, `<navPoint id="chapter0003" playOrder="4">`)
	l4 = strings.Index(ncx, `<navPoint id="chapter0004" playOrder="5">`)
	nextL1 = strings.Index(ncx, `<navPoint id="chapter0005" playOrder="6">`)
	if l2 < 0 || l3 < l2 || l4 < l3 || nextL1 < l4 {
		t.Fatalf("toc.ncx should preserve deep chapter order:\n%s", ncx)
	}
}

func assertZipEntry(t *testing.T, path, name string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	for _, zf := range zr.File {
		if zf.Name == name {
			return
		}
	}
	t.Fatalf("zip entry %s not found", name)
}

func assertZipEntryMissing(t *testing.T, path, name string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	for _, zf := range zr.File {
		if zf.Name == name {
			t.Fatalf("zip entry %s should not exist", name)
		}
	}
}

func readZipEntry(t *testing.T, path, name string) string {
	return string(readZipEntryBytes(t, path, name))
}

func readZipEntryBytes(t *testing.T, path, name string) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		t.Fatal(err)
	}
	for _, zf := range zr.File {
		if zf.Name != name {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	t.Fatalf("zip entry %s not found", name)
	return nil
}
