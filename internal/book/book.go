package book

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"ezpub/internal/lang"
)

type Book struct {
	Title       string
	Author      string
	Translator  string
	Language    string
	Description string
	Publisher   string
	Subjects    []string
	Date        string
	Identifier  string
	Rights      string
	Chapters    []Chapter
	CSS         string
	CoverPath   string
	CoverPage   bool
	Cover       CoverOptions
	FlowSizeKB  int
	Fonts       []string
	FontSources []string
	Images      []string
}

type Chapter struct {
	ID      string
	Title   string
	Level   int
	TOCOnly bool
	Blocks  []Block
}

type Block struct {
	RawHTML bool
	Text    string
}

type LayoutOptions struct {
	LineHeightPercent int
	FontSizePercent   int
	IndentEm          float64
	TextAlign         int
	MarginTop         CSSLength
	MarginBottom      CSSLength
	MarginLeft        CSSLength
	MarginRight       CSSLength
	ParagraphSpacing  CSSLength
}

type CSSLength struct {
	Value float64
	Unit  string
}

type CoverOptions struct {
	Style          int
	TitleFontPX    int
	AuthorFontPX   int
	ScreenHeightPX int
}

type TextOptions struct {
	Title               string
	Author              string
	Translator          string
	Language            string
	Description         string
	Publisher           string
	Subjects            []string
	Date                string
	Identifier          string
	Rights              string
	ChapterRegex        string
	ChapterPlan         []ChapterMark
	FixedChapterLen     int
	SplitCount          int
	RemoveBlankLine     bool
	AddSpace            bool
	AddSpaceCount       int
	SkipEmptyChapters   bool
	EmptyAsSubdirectory bool
	RawHTMLTag          string
	EnableRawHTML       bool
	TOCSpace            bool
	Layout              LayoutOptions
	CustomCSS           string
	CSSMode             int
	CoverPath           string
	Cover               CoverOptions
	SkipCoverPage       bool
	FlowSizeKB          int
	Fonts               []string
	FontSources         []string
	Images              []string
	ImageResolver       map[string]string
}

func FromText(text string, opts TextOptions) (*Book, error) {
	if opts.Title == "" {
		opts.Title = lang.DefaultBookTitle
	}
	if opts.Language == "" {
		opts.Language = lang.DefaultLanguage
	}

	chapters, err := SplitChapters(text, opts)
	if err != nil {
		return nil, err
	}
	if len(chapters) == 0 {
		chapters = []Chapter{{
			ID:     "chapter0001",
			Title:  opts.Title,
			Level:  1,
			Blocks: paragraphs(text, opts),
		}}
	}
	if opts.SkipEmptyChapters {
		chapters = nonEmptyChapters(chapters)
	} else if opts.EmptyAsSubdirectory {
		markEmptyChaptersAsTOCOnly(chapters)
	}
	if len(chapters) == 0 {
		chapters = []Chapter{{
			ID:     "chapter0001",
			Title:  opts.Title,
			Level:  1,
			Blocks: paragraphs(text, opts),
		}}
	}
	for i := range chapters {
		chapters[i].ID = fmt.Sprintf("chapter%04d", i+1)
		if chapters[i].Level < 1 {
			chapters[i].Level = 1
		}
	}

	images := cleanPaths(opts.Images)
	if len(opts.ImageResolver) > 0 {
		for i := range chapters {
			for j := range chapters[i].Blocks {
				if !chapters[i].Blocks[j].RawHTML {
					continue
				}
				html, found := rewriteImageRefs(chapters[i].Blocks[j].Text, opts.ImageResolver)
				chapters[i].Blocks[j].Text = html
				images = cleanPaths(append(images, found...))
			}
		}
	}

	fonts := cleanPaths(opts.Fonts)
	css := DefaultCSS(opts.Layout)
	switch opts.CSSMode {
	case 1:
		css += "\n" + opts.CustomCSS
	case 2:
		if strings.TrimSpace(opts.CustomCSS) != "" {
			css = opts.CustomCSS
		}
	}
	fontDefs, embeddedFamilies := fontCSS(fonts)
	externalDefs, externalFamilies := externalFontCSS(opts.FontSources)
	fontFamilies := append(externalFamilies, embeddedFamilies...)
	css = applyBodyFontFamily(css, fontFamilies)
	css += fontDefs
	css += externalDefs

	return &Book{
		Title:       opts.Title,
		Author:      strings.Join(splitCommaList(opts.Author), ", "),
		Translator:  opts.Translator,
		Language:    opts.Language,
		Description: opts.Description,
		Publisher:   opts.Publisher,
		Subjects:    cleanStrings(opts.Subjects),
		Date:        opts.Date,
		Identifier:  opts.Identifier,
		Rights:      opts.Rights,
		Chapters:    chapters,
		CSS:         css,
		CoverPath:   opts.CoverPath,
		CoverPage:   opts.CoverPath != "" || !opts.SkipCoverPage,
		Cover:       defaultCoverOptions(opts.Cover),
		FlowSizeKB:  opts.FlowSizeKB,
		Fonts:       fonts,
		FontSources: cleanStrings(opts.FontSources),
		Images:      images,
	}, nil
}

func nonEmptyChapters(chapters []Chapter) []Chapter {
	var out []Chapter
	for _, ch := range chapters {
		if chapterHasContent(ch) {
			out = append(out, ch)
		}
	}
	return out
}

func markEmptyChaptersAsTOCOnly(chapters []Chapter) {
	for i := range chapters {
		if !chapterHasContent(chapters[i]) {
			chapters[i].TOCOnly = true
		}
	}
}

func chapterHasContent(ch Chapter) bool {
	for _, block := range ch.Blocks {
		if strings.TrimSpace(block.Text) != "" {
			return true
		}
	}
	return false
}

func SplitChapters(text string, opts TextOptions) ([]Chapter, error) {
	if len(opts.ChapterPlan) > 0 {
		return SplitChaptersByPlan(text, opts.ChapterPlan, opts)
	}
	if opts.SplitCount > 0 {
		return SplitChaptersEven(text, opts.SplitCount, opts), nil
	}
	if opts.FixedChapterLen > 0 {
		return SplitChaptersFixed(text, opts.FixedChapterLen, opts), nil
	}

	lines := splitLines(text)
	if strings.TrimSpace(opts.ChapterRegex) == "" {
		return []Chapter{{
			Title:  opts.Title,
			Level:  1,
			Blocks: paragraphsFromLines(lines, opts),
		}}, nil
	}

	re, err := regexp.Compile(opts.ChapterRegex)
	if err != nil {
		return nil, err
	}

	var chapters []Chapter
	var current *Chapter
	var preface []string

	for _, line := range lines {
		if re.MatchString(line) {
			if current != nil {
				chapters = append(chapters, *current)
			} else if len(nonBlank(preface)) > 0 {
				chapters = append(chapters, Chapter{
					Title:  lang.PrefaceTitle,
					Level:  1,
					Blocks: paragraphsFromLines(preface, opts),
				})
			}
			current = &Chapter{
				Title: strings.TrimSpace(line),
				Level: chapterLevel(line, opts.TOCSpace),
			}
			continue
		}
		if current == nil {
			preface = append(preface, line)
		} else {
			current.Blocks = append(current.Blocks, blockFromLine(line, opts)...)
		}
	}

	if current != nil {
		chapters = append(chapters, *current)
	} else if len(nonBlank(preface)) > 0 {
		chapters = append(chapters, Chapter{
			Title:  opts.Title,
			Level:  1,
			Blocks: paragraphsFromLines(preface, opts),
		})
	}

	return chapters, nil
}

func DefaultCSS(layout LayoutOptions) string {
	if layout.LineHeightPercent == 0 {
		layout.LineHeightPercent = 130
	}
	if layout.FontSizePercent == 0 {
		layout.FontSizePercent = 100
	}
	layout.MarginTop = defaultCSSLength(layout.MarginTop, "0", "em")
	layout.MarginBottom = defaultCSSLength(layout.MarginBottom, "0", "em")
	layout.MarginLeft = defaultCSSLength(layout.MarginLeft, "0", "em")
	layout.MarginRight = defaultCSSLength(layout.MarginRight, "0", "em")
	layout.ParagraphSpacing = defaultCSSLength(layout.ParagraphSpacing, ".35", "em")

	align := "inherit"
	switch layout.TextAlign {
	case 1:
		align = "left"
	case 2:
		align = "center"
	case 3:
		align = "right"
	case 4:
		align = "justify"
	}

	return fmt.Sprintf(`/*  Generated by ezPub  */
/*  Default CSS follows EasyPub v1.50 layout classes.  */

@page {
  margin-top: %s;
  margin-bottom: %s;
}

body {
  padding: 0;
  margin-left: %s;
  margin-right: %s;
  orphans: 0;
  widows: 0;
}

p {
  font-size: %d%%;
  line-height: %d%%;
  margin-top: %s;
  margin-bottom: 0;
  margin-left: 0;
  margin-right: 0;
  text-align: %s;
  orphans: 0;
  widows: 0;
}

.a {
  text-indent: %.3gem;
}

div.centeredimage {
  text-align: center;
  display: block;
  margin-top: 0.5em;
  margin-bottom: 0.5em;
}
img.attpic {
  border: 1px solid #000000;
  max-width: 100%%;
  margin: 0;
}

.booktitle {
  margin-top: 30%%;
  margin-bottom: 0;
  border-style: none solid none none;
  border-width: 50px;
  border-color: #4E594D;
  font-size: 3em;
  line-height: 120%%;
  text-align: right;
}
.bookauthor {
  margin-top: 0;
  border-style: none solid none none;
  border-width: 50px;
  border-color: #4E594D;
  page-break-after: always;
  font-size: large;
  line-height: 120%%;
  text-align: right;
}

.titletoc, .titlel1top, .titlel1std,.titlel2top, .titlel2std,.titlel3top, .titlel3std,.titlel4std {
  margin-top: 0;
  border-style: none double none solid;
  border-width: 0px 5px 0px 20px;
  border-color: #586357;
  background-color: #C1CCC0;
  padding: 45px 5px 5px 5px;
  font-size: x-large;
  line-height: 115%%;
  text-align: justify;
}

.titlel1single,.titlel2single,.titlel3single {
  margin-top: 35%%;
  border-style: none solid none none;
  border-width: 30px;
  border-color: #4E594D;
  padding: 30px 5px 5px 5px;
  font-size: x-large;
  line-height: 125%%;
  text-align: right;
}

.toc {
  margin-left: 16%%;
  padding: 0px;
  line-height: 130%%;
  text-align: justify;
}

.toc a {
  text-decoration: none;
  color: #000000;
}

.toc ol {
  list-style: none;
  margin: 0;
  padding: 0;
}

.toc ol ol {
  margin: 0;
  padding: 0;
}

.tocl1 {
  margin-top: 0.5em;
  margin-left: -30px;
  border-style: none double double solid;
  border-width: 0px 5px 2px 20px;
  border-color: #6B766A;
  line-height: 135%%;
  font-size: 132%%;
}

.tocl2 {
  margin-top: 0.5em;
  margin-left: -20px;
  border-style: none double none solid;
  border-width: 0px 2px 0px 10px;
  border-color: #939E92;
  line-height: 123%%;
  font-size: 120%%;
}

.tocl3 {
  margin-top: 0.5em;
  margin-left: -20px;
  border-style: none double none solid;
  border-width: 0px 2px 0px 8px;
  border-color: #939E92;
  line-height: 112%%;
  font-size: 109%%;
}

.tocl4 {
  margin-top: 0.5em;
  margin-left: -20px;
  border-style: none double none solid;
  border-width: 0px 2px 0px 6px;
  border-color: #939E92;
  line-height: 115%%;
  font-size: 110%%;
}

.subtoc {
  margin-left: 15%%;
  padding: 0px;
  text-align: justify;
}

.subtoclist {
  margin-top: 0.5em;
  margin-left: -20px;
  border-style: none double none solid;
  border-width: 0px 2px 0px 10px;
  border-color: #939E92;
  line-height: 123%%;
  font-size: 120%%;
}

.cover {
  text-align: center;
}
.cover-page,
.cover-page .cover {
  height: 100%%;
  margin: 0;
  padding: 0;
}
.cover table {
  width: 100%%;
  height: 100%%;
  border-collapse: collapse;
}
.cover td {
  text-align: center;
  vertical-align: middle;
}
.cover img {
  max-width: 100%%;
  max-height: 100%%;
}`, layout.MarginTop.String(), layout.MarginBottom.String(), layout.MarginLeft.String(), layout.MarginRight.String(), layout.FontSizePercent, layout.LineHeightPercent, layout.ParagraphSpacing.String(), align, layout.IndentEm)
}

func defaultCoverOptions(cover CoverOptions) CoverOptions {
	if cover.TitleFontPX == 0 {
		cover.TitleFontPX = 50
	}
	if cover.AuthorFontPX == 0 {
		cover.AuthorFontPX = 25
	}
	if cover.ScreenHeightPX == 0 {
		cover.ScreenHeightPX = 720
	}
	return cover
}

func defaultCSSLength(length CSSLength, fallbackValue, fallbackUnit string) CSSLength {
	if length.Unit == "" {
		length.Unit = fallbackUnit
	}
	if length.Value == 0 && fallbackValue != "0" {
		if fallbackValue == ".35" {
			length.Value = .35
		}
	}
	return length
}

func (l CSSLength) String() string {
	unit := strings.TrimSpace(l.Unit)
	if unit == "" {
		unit = "em"
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", l.Value), "0"), ".") + unit
}

func paragraphs(text string, opts TextOptions) []Block {
	return paragraphsFromLines(splitLines(text), opts)
}

func paragraphsFromLines(lines []string, opts TextOptions) []Block {
	var blocks []Block
	for _, line := range lines {
		blocks = append(blocks, blockFromLine(line, opts)...)
	}
	return blocks
}

func blockFromLine(line string, opts TextOptions) []Block {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		if opts.RemoveBlankLine {
			return nil
		}
		return []Block{{Text: ""}}
	}
	if opts.EnableRawHTML && opts.RawHTMLTag != "" && strings.HasPrefix(trimmed, opts.RawHTMLTag) {
		return []Block{{RawHTML: true, Text: strings.TrimPrefix(trimmed, opts.RawHTMLTag)}}
	}
	if opts.AddSpace && opts.AddSpaceCount > 0 {
		trimmed = strings.Repeat("　", opts.AddSpaceCount) + trimmed
	}
	return []Block{{Text: trimmed}}
}

func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}

func chapterLevel(line string, tocSpace bool) int {
	if !tocSpace {
		return 1
	}
	level := 1
	for _, r := range line {
		switch r {
		case ' ', '\t', '　':
			level++
		default:
			if level > 6 {
				return 6
			}
			return level
		}
	}
	return 1
}

func nonBlank(lines []string) []string {
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func cleanPaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		clean := filepath.Clean(p)
		key := strings.ToLower(clean)
		if !seen[key] {
			seen[key] = true
			out = append(out, clean)
		}
	}
	return out
}

func fontCSS(fonts []string) (string, []string) {
	if len(fonts) == 0 {
		return "", nil
	}
	var b strings.Builder
	var families []string
	b.WriteString("\n")
	for i, fontPath := range fonts {
		name := filepath.Base(fontPath)
		family := fmt.Sprintf("ezpub-font-%d", i+1)
		families = append(families, family)
		b.WriteString(fmt.Sprintf("@font-face { font-family: %q; src: url(\"../fonts/%s\"); }\n", family, name))
	}
	return b.String(), families
}

func externalFontCSS(sources []string) (string, []string) {
	sources = cleanStrings(sources)
	if len(sources) == 0 {
		return "", nil
	}
	var urls []string
	for _, source := range sources {
		urls = append(urls, fmt.Sprintf("url(\"%s\")", source))
	}
	return fmt.Sprintf("\n@font-face { font-family: \"ezpub-reader-font\"; src: %s; }\n", strings.Join(urls, ", ")), []string{"ezpub-reader-font"}
}

func applyBodyFontFamily(css string, families []string) string {
	families = cleanStrings(families)
	if len(families) == 0 {
		return css
	}
	var quoted []string
	for _, family := range families {
		quoted = append(quoted, fmt.Sprintf("%q", family))
	}
	decl := "  font-family: " + strings.Join(quoted, ", ") + ", serif;\n"
	bodyRe := regexp.MustCompile(`(?m)body\s*\{`)
	loc := bodyRe.FindStringIndex(css)
	if loc == nil {
		return strings.TrimRight(css, "\n") + "\nbody {\n" + decl + "}\n"
	}
	bodyStart := loc[1]
	bodyEndRel := strings.Index(css[bodyStart:], "}")
	if bodyEndRel < 0 {
		return css[:bodyStart] + "\n" + decl + css[bodyStart:]
	}
	bodyEnd := bodyStart + bodyEndRel
	bodyContent := css[bodyStart:bodyEnd]
	fontRe := regexp.MustCompile(`(?m)^\s*font-family\s*:[^;]+;\s*`)
	if fontLoc := fontRe.FindStringIndex(bodyContent); fontLoc != nil {
		return css[:bodyStart+fontLoc[0]] + decl + css[bodyStart+fontLoc[1]:]
	}
	return css[:bodyStart] + "\n" + decl + css[bodyStart:]
}
