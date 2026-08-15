package epub

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ezpub/internal/book"
	"ezpub/internal/lang"
)

const VerticalCSS = `html {
  writing-mode: vertical-rl;
  -epub-writing-mode: vertical-rl;
  -webkit-writing-mode: vertical-rl;
  line-break: strict;
  -epub-line-break: strict;
  -webkit-line-break: strict;
}`

type resource struct {
	ID         string
	Href       string
	MediaType  string
	Properties string
}

type contentDoc struct {
	ID      string
	Href    string
	Chapter book.Chapter
}

type tocNode struct {
	ch       book.Chapter
	level    int
	children []*tocNode
}

func Write(path string, bk *book.Book) error {
	if bk == nil {
		return fmt.Errorf(lang.ErrNilBook)
	}
	if len(bk.Chapters) == 0 {
		return fmt.Errorf(lang.ErrEmptyBook)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()

	id := "urn:uuid:" + randomHex(16)
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	if err := addStored(zw, "mimetype", []byte("application/epub+zip")); err != nil {
		return err
	}
	if err := addText(zw, "META-INF/container.xml", containerXML()); err != nil {
		return err
	}
	if err := addText(zw, "EPUB/styles/style.css", bk.CSS); err != nil {
		return err
	}

	resources := []resource{
		{ID: "nav", Href: "text/nav.xhtml", MediaType: "application/xhtml+xml", Properties: "nav"},
		{ID: "ncx", Href: "toc.ncx", MediaType: "application/x-dtbncx+xml"},
		{ID: "htmltoc", Href: "text/book-toc.xhtml", MediaType: "application/xhtml+xml"},
		{ID: "css", Href: "styles/style.css", MediaType: "text/css"},
	}
	if bk.CoverPage {
		resources = append(resources, resource{ID: "cover-page", Href: "text/cover.xhtml", MediaType: "application/xhtml+xml"})
	}

	coverImageHref := ""
	if bk.CoverPath != "" {
		name := "images/cover" + strings.ToLower(filepath.Ext(bk.CoverPath))
		if err := addFile(zw, "EPUB/"+name, bk.CoverPath); err != nil {
			return err
		}
		coverImageHref = name
		resources = append(resources, resource{
			ID:         "cover-image",
			Href:       name,
			MediaType:  mediaTypeFor(name),
			Properties: "cover-image",
		})
	}

	for i, fontPath := range bk.Fonts {
		name := "fonts/" + filepath.Base(fontPath)
		if err := addFile(zw, "EPUB/"+name, fontPath); err != nil {
			return err
		}
		resources = append(resources, resource{
			ID:        fmt.Sprintf("font%d", i+1),
			Href:      name,
			MediaType: mediaTypeFor(name),
		})
	}

	for i, imagePath := range bk.Images {
		name := "images/" + filepath.Base(imagePath)
		if err := addFile(zw, "EPUB/"+name, imagePath); err != nil {
			return err
		}
		resources = append(resources, resource{
			ID:        fmt.Sprintf("image%d", i+1),
			Href:      name,
			MediaType: mediaTypeFor(name),
		})
	}

	docs := chapterDocs(bk)
	for _, doc := range docs {
		if err := addText(zw, "EPUB/"+doc.Href, chapterXHTML(bk, doc.Chapter)); err != nil {
			return err
		}
		resources = append(resources, resource{
			ID:        doc.ID,
			Href:      doc.Href,
			MediaType: "application/xhtml+xml",
		})
	}

	if bk.CoverPage {
		if err := addText(zw, "EPUB/text/cover.xhtml", coverXHTML(bk, coverImageHref)); err != nil {
			return err
		}
	}
	if err := addText(zw, "EPUB/text/nav.xhtml", navXHTML(bk)); err != nil {
		return err
	}
	if err := addText(zw, "EPUB/text/book-toc.xhtml", bookTOCXHTML(bk)); err != nil {
		return err
	}
	if err := addText(zw, "EPUB/toc.ncx", tocNCX(bk, resolvedBookID(bk, id))); err != nil {
		return err
	}
	if err := addText(zw, "EPUB/content.opf", contentOPF(bk, id, now, resources, docs)); err != nil {
		return err
	}

	return nil
}

func containerXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="EPUB/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
`
}

func contentOPF(bk *book.Book, id, modified string, resources []resource, docs []contentDoc) string {
	bookID := resolvedBookID(bk, id)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="bookid" prefix="rendition: http://www.idpf.org/vocab/rendition/#">` + "\n")
	b.WriteString("  <metadata xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n")
	b.WriteString(fmt.Sprintf("    <dc:identifier id=\"bookid\">%s</dc:identifier>\n", escape(bookID)))
	b.WriteString(fmt.Sprintf("    <dc:title>%s</dc:title>\n", escape(bk.Title)))
	for _, author := range splitCommaList(bk.Author) {
		b.WriteString(fmt.Sprintf("    <dc:creator>%s</dc:creator>\n", escape(author)))
	}
	if bk.Translator != "" {
		b.WriteString(fmt.Sprintf("    <dc:contributor>%s</dc:contributor>\n", escape(bk.Translator)))
	}
	if bk.Description != "" {
		b.WriteString(fmt.Sprintf("    <dc:description>%s</dc:description>\n", escape(bk.Description)))
	}
	if bk.Publisher != "" {
		b.WriteString(fmt.Sprintf("    <dc:publisher>%s</dc:publisher>\n", escape(bk.Publisher)))
	}
	for _, subject := range bk.Subjects {
		b.WriteString(fmt.Sprintf("    <dc:subject>%s</dc:subject>\n", escape(subject)))
	}
	if bk.Series != "" {
		b.WriteString(fmt.Sprintf("    <meta property=\"belongs-to-collection\" id=\"series\">%s</meta>\n", escape(bk.Series)))
		b.WriteString("    <meta property=\"collection-type\" refines=\"#series\">series</meta>\n")
		if bk.SeriesIndex > 0 {
			seriesIndex := strconv.FormatFloat(bk.SeriesIndex, 'f', -1, 64)
			b.WriteString(fmt.Sprintf("    <meta property=\"group-position\" refines=\"#series\">%s</meta>\n", escape(seriesIndex)))
			b.WriteString(fmt.Sprintf("    <meta name=\"calibre:series_index\" content=\"%s\"/>\n", escape(seriesIndex)))
		}
		b.WriteString(fmt.Sprintf("    <meta name=\"calibre:series\" content=\"%s\"/>\n", escape(bk.Series)))
	}
	if bk.Date != "" {
		b.WriteString(fmt.Sprintf("    <dc:date>%s</dc:date>\n", escape(bk.Date)))
	}
	if bk.Rights != "" {
		b.WriteString(fmt.Sprintf("    <dc:rights>%s</dc:rights>\n", escape(bk.Rights)))
	}
	lang := bk.Language
	if lang == "" {
		lang = "zh-TW"
	}
	b.WriteString(fmt.Sprintf("    <dc:language>%s</dc:language>\n", escape(lang)))
	b.WriteString(fmt.Sprintf("    <meta property=\"dcterms:modified\">%s</meta>\n", escape(modified)))
	if bk.CoverPath != "" {
		b.WriteString("    <meta name=\"cover\" content=\"cover-image\"/>\n")
	}
	b.WriteString("  </metadata>\n")
	b.WriteString("  <manifest>\n")
	for _, r := range resources {
		props := ""
		if r.Properties != "" {
			props = fmt.Sprintf(" properties=\"%s\"", escape(r.Properties))
		}
		b.WriteString(fmt.Sprintf("    <item id=\"%s\" href=\"%s\" media-type=\"%s\"%s/>\n", escape(r.ID), escape(r.Href), escape(r.MediaType), props))
	}
	b.WriteString("  </manifest>\n")
	b.WriteString("  <spine toc=\"ncx\">\n")
	if bk.CoverPage {
		b.WriteString("    <itemref idref=\"cover-page\"/>\n")
	}
	b.WriteString("    <itemref idref=\"htmltoc\"/>\n")
	for _, doc := range docs {
		b.WriteString(fmt.Sprintf("    <itemref idref=\"%s\"/>\n", escape(doc.ID)))
	}
	b.WriteString("  </spine>\n")
	if bk.CoverPage || len(bk.Chapters) > 0 {
		b.WriteString("  <guide>\n")
		if bk.CoverPage {
			b.WriteString("    <reference type=\"cover\" title=\"Cover\" href=\"text/cover.xhtml\"/>\n")
		}
		b.WriteString("    <reference type=\"toc\" title=\"Table Of Contents\" href=\"text/book-toc.xhtml\"/>\n")
		if first := firstReadableChapterID(bk); first != "" {
			b.WriteString("    <reference type=\"text\" title=\"Beginning\" href=\"text/" + escape(first) + ".xhtml\"/>\n")
		}
		b.WriteString("  </guide>\n")
	}
	b.WriteString("</package>\n")
	return b.String()
}

func resolvedBookID(bk *book.Book, fallback string) string {
	if bk != nil && strings.TrimSpace(bk.Identifier) != "" {
		return strings.TrimSpace(bk.Identifier)
	}
	return fallback
}

func chapterDocs(bk *book.Book) []contentDoc {
	var docs []contentDoc
	maxBytes := bk.FlowSizeKB * 1024
	for _, ch := range bk.Chapters {
		if ch.TOCOnly {
			continue
		}
		parts := splitChapterBySize(ch, maxBytes)
		for i, part := range parts {
			id := ch.ID
			if i > 0 {
				id = fmt.Sprintf("%s_%04d", ch.ID, i+1)
			}
			part.ID = id
			docs = append(docs, contentDoc{
				ID:      id,
				Href:    "text/" + id + ".xhtml",
				Chapter: part,
			})
		}
	}
	return docs
}

func splitChapterBySize(ch book.Chapter, maxBytes int) []book.Chapter {
	if maxBytes <= 0 || approxChapterBytes(ch.Blocks) <= maxBytes {
		return []book.Chapter{ch}
	}
	var parts []book.Chapter
	var blocks []book.Block
	size := 0
	for _, block := range ch.Blocks {
		blockSize := len(block.Text) + 32
		if len(blocks) > 0 && size+blockSize > maxBytes {
			parts = append(parts, book.Chapter{Title: ch.Title, Level: ch.Level, Blocks: blocks})
			blocks = nil
			size = 0
		}
		blocks = append(blocks, block)
		size += blockSize
	}
	if len(blocks) > 0 || len(parts) == 0 {
		parts = append(parts, book.Chapter{Title: ch.Title, Level: ch.Level, Blocks: blocks})
	}
	return parts
}

func approxChapterBytes(blocks []book.Block) int {
	size := 0
	for _, block := range blocks {
		size += len(block.Text) + 32
	}
	return size
}

func navXHTML(bk *book.Book) string {
	var b strings.Builder
	b.WriteString(xhtmlHead(bk.Title, "../styles/style.css"))
	b.WriteString(fmt.Sprintf("<body>\n<nav epub:type=\"toc\" id=\"toc\"><h2 class=\"titletoc\">%s</h2>\n", escape(lang.TOCTitle)))
	writeNavXHTMLItems(&b, buildTOCTree(bk.Chapters))
	b.WriteString("</nav>\n</body>\n</html>\n")
	return b.String()
}

func bookTOCXHTML(bk *book.Book) string {
	var b strings.Builder
	b.WriteString(xhtmlHead("Table Of Contents", "../styles/style.css"))
	b.WriteString("<body>\n")
	b.WriteString(fmt.Sprintf("<h2 class=\"titletoc\">%s</h2>\n", escape(lang.TOCTitle)))
	b.WriteString("<div class=\"toc\">\n<dl>\n")
	for _, ch := range bk.Chapters {
		level := easyPubTOCLevel(ch.Level)
		class := fmt.Sprintf("tocl%d", level)
		target := ch.ID
		if ch.TOCOnly {
			target = nextReadableChapterID(bk, ch.ID)
		}
		if target == "" {
			b.WriteString(fmt.Sprintf("<dt class=\"%s\"><span>%s</span></dt>\n", class, escape(ch.Title)))
			b.WriteString("<dd></dd>\n")
			continue
		}
		b.WriteString(fmt.Sprintf("<dt class=\"%s\"><a href=\"%s.xhtml\">%s</a></dt>\n", class, escape(target), escape(ch.Title)))
		b.WriteString("<dd></dd>\n")
	}
	b.WriteString("</dl>\n</div>\n</body>\n</html>\n")
	return b.String()
}

func buildTOCTree(chapters []book.Chapter) []*tocNode {
	roots := make([]*tocNode, 0, len(chapters))
	var stack []*tocNode
	for _, ch := range chapters {
		level := ch.Level
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		node := &tocNode{ch: ch, level: level}
		for len(stack) >= level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1]
			parent.children = append(parent.children, node)
		}
		stack = append(stack, node)
	}
	return roots
}

func writeNavXHTMLItems(b *strings.Builder, nodes []*tocNode) {
	b.WriteString("<ol class=\"toc\">\n")
	for _, node := range nodes {
		class := fmt.Sprintf(" class=\"tocl%d\"", easyPubTOCLevel(node.level))
		b.WriteString(fmt.Sprintf("<li%s>", class))
		if node.ch.TOCOnly {
			b.WriteString(fmt.Sprintf("<span>%s</span>", escape(node.ch.Title)))
		} else {
			b.WriteString(fmt.Sprintf("<a href=\"%s.xhtml\">%s</a>", escape(node.ch.ID), escape(node.ch.Title)))
		}
		if len(node.children) > 0 {
			b.WriteByte('\n')
			writeNavXHTMLItems(b, node.children)
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ol>\n")
}

func tocNCX(bk *book.Book, id string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">` + "\n")
	b.WriteString("  <head>\n")
	if bk.CoverPage {
		b.WriteString("    <meta name=\"cover\" content=\"cover-page\"/>\n")
	}
	b.WriteString(fmt.Sprintf("    <meta name=\"dtb:uid\" content=\"%s\"/>\n", escape(id)))
	b.WriteString(fmt.Sprintf("    <meta name=\"dtb:depth\" content=\"%d\"/>\n", tocDepth(bk.Chapters)))
	b.WriteString("    <meta name=\"dtb:totalPageCount\" content=\"0\"/>\n")
	b.WriteString("    <meta name=\"dtb:maxPageNumber\" content=\"0\"/>\n")
	b.WriteString("  </head>\n")
	b.WriteString(fmt.Sprintf("  <docTitle><text>%s</text></docTitle>\n", escape(bk.Title)))
	if bk.Author != "" {
		b.WriteString(fmt.Sprintf("  <docAuthor><text>%s</text></docAuthor>\n", escape(bk.Author)))
	}
	b.WriteString("  <navMap>\n")
	playOrder := 1
	if bk.CoverPage {
		b.WriteString(fmt.Sprintf("    <navPoint id=\"cover\" playOrder=\"%d\">\n", playOrder))
		b.WriteString("      <navLabel><text>封面</text></navLabel>\n")
		b.WriteString("      <content src=\"text/cover.xhtml\"/>\n")
		b.WriteString("    </navPoint>\n")
		playOrder++
	}
	b.WriteString(fmt.Sprintf("    <navPoint id=\"htmltoc\" playOrder=\"%d\">\n", playOrder))
	b.WriteString(fmt.Sprintf("      <navLabel><text>%s</text></navLabel>\n", escape(lang.TOCTitle)))
	b.WriteString("      <content src=\"text/book-toc.xhtml\"/>\n")
	b.WriteString("    </navPoint>\n")
	playOrder++
	writeNCXItems(&b, buildTOCTree(bk.Chapters), bk, &playOrder, 2)
	b.WriteString("  </navMap>\n")
	b.WriteString("</ncx>\n")
	return b.String()
}

func tocDepth(chapters []book.Chapter) int {
	depth := 1
	for _, ch := range chapters {
		if ch.Level > depth {
			depth = ch.Level
		}
	}
	if depth > 6 {
		return 6
	}
	if depth < 1 {
		return 1
	}
	return depth
}

func writeNCXItems(b *strings.Builder, nodes []*tocNode, bk *book.Book, playOrder *int, indent int) {
	prefix := strings.Repeat("  ", indent)
	for _, node := range nodes {
		id := node.ch.ID
		if id == "" {
			id = fmt.Sprintf("navPoint-%d", *playOrder)
		}
		b.WriteString(fmt.Sprintf("%s<navPoint id=\"%s\" playOrder=\"%d\">\n", prefix, escape(id), *playOrder))
		b.WriteString(fmt.Sprintf("%s  <navLabel><text>%s%s</text></navLabel>\n", prefix, strings.Repeat("　", maxInt(0, node.level-1)), escape(node.ch.Title)))
		target := node.ch.ID
		if node.ch.TOCOnly {
			target = nextReadableChapterID(bk, node.ch.ID)
		}
		if target != "" {
			b.WriteString(fmt.Sprintf("%s  <content src=\"text/%s.xhtml\"/>\n", prefix, escape(target)))
		}
		*playOrder++
		writeNCXItems(b, node.children, bk, playOrder, indent+1)
		b.WriteString(fmt.Sprintf("%s</navPoint>\n", prefix))
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func firstReadableChapterID(bk *book.Book) string {
	if bk == nil {
		return ""
	}
	for _, ch := range bk.Chapters {
		if !ch.TOCOnly {
			return ch.ID
		}
	}
	return ""
}

func nextReadableChapterID(bk *book.Book, afterID string) string {
	if bk == nil {
		return ""
	}
	seen := false
	for _, ch := range bk.Chapters {
		if ch.ID == afterID {
			seen = true
			continue
		}
		if seen && !ch.TOCOnly {
			return ch.ID
		}
	}
	return firstReadableChapterID(bk)
}

func coverXHTML(bk *book.Book, coverImageHref string) string {
	var b strings.Builder
	cover := bk.Cover
	if cover.ScreenHeightPX == 0 {
		cover.ScreenHeightPX = 720
	}
	b.WriteString(xhtmlHead(bk.Title, "../styles/style.css"))
	b.WriteString("<body class=\"cover-page\">\n<section class=\"cover\">\n")
	if coverImageHref != "" {
		halfHeight := cover.ScreenHeightPX / 2
		b.WriteString(fmt.Sprintf("<div class=\"wedge\" style=\"float:left;height:50%%;margin-bottom:-%dpx\"></div>\n", halfHeight))
		b.WriteString("<div class=\"container\" style=\"clear:both;height:0;position:relative\">\n")
		b.WriteString(fmt.Sprintf("<table style=\"height:%dpx;width:100%%;text-align:center\"><tr><td>\n", cover.ScreenHeightPX))
		b.WriteString(fmt.Sprintf("<img src=\"../%s\" alt=\"%s\"/>\n", escape(coverImageHref), escape(bk.Title)))
		b.WriteString("</td></tr></table>\n</div>\n")
	} else {
		b.WriteString("<div>\n")
		b.WriteString(fmt.Sprintf("<h1 class=\"booktitle\">%s</h1>\n", escape(bk.Title)))
		if bk.Author != "" {
			b.WriteString(fmt.Sprintf("<h3 class=\"bookauthor\">%s</h3>\n", escape(bk.Author)))
		}
		b.WriteString("</div>\n")
	}
	b.WriteString("</section>\n</body>\n</html>\n")
	return b.String()
}

func chapterXHTML(bk *book.Book, ch book.Chapter) string {
	var b strings.Builder
	b.WriteString(xhtmlHead(ch.Title+" - "+bk.Title, "../styles/style.css"))
	b.WriteString("<body>\n")
	b.WriteString(fmt.Sprintf("<h2 id=\"title\" class=\"titlel%dstd\">%s</h2>\n", easyPubTOCLevel(ch.Level), escape(ch.Title)))
	for _, block := range ch.Blocks {
		if block.RawHTML {
			b.WriteString(block.Text)
			b.WriteByte('\n')
			continue
		}
		b.WriteString(fmt.Sprintf("<p class=\"a\">%s</p>\n", escape(block.Text)))
	}
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

func easyPubTOCLevel(level int) int {
	if level < 1 {
		level = 1
	}
	level++
	if level > 4 {
		return 4
	}
	return level
}

func xhtmlHead(title, cssHref string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head>
  <meta charset="utf-8"/>
  <title>%s</title>
  <link rel="stylesheet" type="text/css" href="%s"/>
</head>
`, escape(title), escape(cssHref))
}

func addStored(zw *zip.Writer, name string, data []byte) error {
	h := &zip.FileHeader{Name: name, Method: zip.Store}
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func addText(zw *zip.Writer, name, text string) error {
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, strings.NewReader(text))
	return err
}

func addFile(zw *zip.Writer, name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	h := &zip.FileHeader{Name: filepath.ToSlash(name), Method: zip.Deflate}
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, bytes.NewReader(data))
	return err
}

func mediaTypeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".css":
		return "text/css"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".xhtml", ".html":
		return "application/xhtml+xml"
	default:
		return "application/octet-stream"
	}
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func escape(s string) string {
	return html.EscapeString(s)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
