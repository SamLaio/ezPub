package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ezpub/internal/book"
	"ezpub/internal/config"
	"ezpub/internal/convert"
	"ezpub/internal/epub"
	"ezpub/internal/lang"
	"ezpub/internal/textio"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ezpub:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return runGUI(nil)
	}

	switch args[0] {
	case "build":
		return runBuild(args[1:])
	case "gui":
		return runGUI(args[1:])
	case "chapters":
		return runChapters(args[1:])
	case "images":
		return runImages(args[1:])
	case "txt":
		return runTXT(args[1:])
	case "inspect-config":
		return runInspectConfig(args[1:])
	case "version", "-v", "--version":
		fmt.Println(lang.Version)
		return nil
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		if strings.HasPrefix(args[0], "-") {
			return errors.New(lang.ErrMissingCommand)
		}
		// Friendly shortcut: ezpub novel.txt -o novel.epub
		return runBuild(args)
	}
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		outPath           string
		configPath        string
		title             string
		author            string
		translator        string
		language          string
		description       string
		publisher         string
		subjects          multiFlag
		category          multiFlag
		date              string
		identifier        string
		isbn              string
		rights            string
		coverPath         string
		cssPath           string
		fonts             multiFlag
		fontSources       multiFlag
		fontType          int
		images            multiFlag
		imageList         string
		scanImages        multiFlag
		encodingName      string
		regex             string
		chapterFile       string
		splitLength       int
		splitCount        int
		vertical          bool
		subsetFonts       bool
		subsetTool        string
		forceTextCover    bool
		removeBlankLine   bool
		addSpace          bool
		addSpaceCount     int
		saveCSS           bool
		cssMode           int
		lineHeight        int
		fontSize          int
		indent            float64
		marginTop         float64
		marginBottom      float64
		marginLeft        float64
		marginRight       float64
		marginUnit        int
		textAlign         int
		rawHTMLTag        string
		enableRawHTML     bool
		tocSpace          bool
		flowSize          int
		emptyChapterStyle int
		coverStyle        int
		titleFont         int
		authorFont        int
		screenHeight      int
	)

	fs.StringVar(&outPath, "o", "", "output path (.epub)")
	fs.StringVar(&configPath, "config", "", "EasyPub config.xml path")
	fs.StringVar(&title, "title", "", "book title")
	fs.StringVar(&author, "author", "", "book author")
	fs.StringVar(&translator, "translator", "", "book translator")
	fs.StringVar(&language, "lang", "zh-TW", "book language")
	fs.StringVar(&description, "description", "", "book description")
	fs.StringVar(&publisher, "publisher", "", "book publisher")
	fs.Var(&subjects, "subject", "book subject; can be repeated")
	fs.Var(&category, "category", "book category; can be repeated")
	fs.StringVar(&date, "date", "", "book date")
	fs.StringVar(&identifier, "identifier", "", "book identifier")
	fs.StringVar(&isbn, "isbn", "", "book ISBN; alias for identifier")
	fs.StringVar(&rights, "rights", "", "book rights")
	fs.StringVar(&coverPath, "cover", "", "cover image path")
	fs.StringVar(&cssPath, "css", "", "custom css path")
	fs.Var(&fonts, "font", "font file path; can be repeated")
	fs.Var(&fontSources, "font-source", "reader/system font source URI; can be repeated")
	fs.IntVar(&fontType, "font-type", -1, "font mode: 0 machine preset, 1 custom reader font, 2 embedded font, 3 reader default")
	fs.Var(&images, "image", "image asset path; can be repeated")
	fs.StringVar(&imageList, "image-list", "", "image TSV from the images command")
	fs.Var(&scanImages, "scan-images", "scan image directory and rewrite raw HTML img src; can be repeated")
	fs.StringVar(&encodingName, "encoding", "auto", "text encoding: auto, utf-8, gbk, gb18030, utf-16le, utf-16be")
	fs.StringVar(&regex, "chapter-regex", "", "chapter title regular expression")
	fs.StringVar(&chapterFile, "chapters", "", "manually edited chapter TSV")
	fs.IntVar(&splitLength, "split-length", 0, "split chapters by roughly n characters")
	fs.IntVar(&splitCount, "split-count", 0, "split into n even chapters, matching EasyPub length split")
	fs.BoolVar(&vertical, "vertical", false, "append vertical writing css")
	fs.BoolVar(&subsetFonts, "subset-fonts", false, "subset embedded fonts with pyftsubset")
	fs.StringVar(&subsetTool, "subset-tool", "", "pyftsubset executable path")
	fs.BoolVar(&forceTextCover, "force-text-cover", false, "create a text cover when no cover image is provided")
	fs.BoolVar(&removeBlankLine, "remove-blank-line", false, "remove blank lines")
	fs.BoolVar(&addSpace, "add-space", false, "add full-width spaces before paragraphs")
	fs.IntVar(&addSpaceCount, "add-space-count", 0, "number of full-width spaces to add")
	fs.BoolVar(&saveCSS, "save-css", false, "save and reload custom CSS from ezPub cache")
	fs.IntVar(&cssMode, "css-mode", -1, "custom CSS mode: 0 skip, 1 append, 2 overwrite")
	fs.IntVar(&lineHeight, "line-height", 0, "line height percent")
	fs.IntVar(&fontSize, "font-size", 0, "font size percent")
	fs.Float64Var(&indent, "indent", -1, "first line indent in em")
	fs.Float64Var(&marginTop, "margin-top", -1, "page top margin")
	fs.Float64Var(&marginBottom, "margin-bottom", -1, "page bottom margin")
	fs.Float64Var(&marginLeft, "margin-left", -1, "page left margin")
	fs.Float64Var(&marginRight, "margin-right", -1, "page right margin")
	fs.IntVar(&marginUnit, "margin-unit", -1, "page margin unit: 0 px, 1 %, 2 em")
	fs.IntVar(&textAlign, "text-align", -1, "text align: 0 default, 1 left, 2 center, 3 right, 4 justify")
	fs.StringVar(&rawHTMLTag, "raw-html-tag", "", "EasyPub raw HTML marker")
	fs.BoolVar(&enableRawHTML, "enable-raw-html", false, "enable EasyPub raw HTML marker")
	fs.BoolVar(&tocSpace, "toc-space", false, "use leading spaces to build nested TOC levels")
	fs.IntVar(&flowSize, "flow-size", -1, "split XHTML flows over this size in KB; 0 disables")
	fs.IntVar(&emptyChapterStyle, "empty-chapter-style", -1, "empty chapter style: 0 skip, 1 create, 2 TOC-only subdirectory")
	fs.IntVar(&coverStyle, "cover-style", -1, "cover image style")
	fs.IntVar(&titleFont, "cover-title-font", 0, "text cover title font size in px")
	fs.IntVar(&authorFont, "cover-author-font", 0, "text cover author font size in px")
	fs.IntVar(&screenHeight, "screen-height", 0, "cover screen height in px")

	if err := fs.Parse(interspersedFlags(args, map[string]bool{
		"vertical":          true,
		"subset-fonts":      true,
		"force-text-cover":  true,
		"remove-blank-line": true,
		"add-space":         true,
		"save-css":          true,
		"enable-raw-html":   true,
		"toc-space":         true,
	})); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New(lang.ErrBuildInputCount)
	}
	splitCountFlag := hasFlag(args, "split-count")

	inputPath := fs.Arg(0)
	cfg, cfgPath, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	applyBuildOverrides(cfg, args, buildOverrides{
		forceTextCover:    forceTextCover,
		removeBlankLine:   removeBlankLine,
		addSpace:          addSpace,
		addSpaceCount:     addSpaceCount,
		saveCSS:           saveCSS,
		cssMode:           cssMode,
		lineHeight:        lineHeight,
		fontSize:          fontSize,
		indent:            indent,
		marginTop:         marginTop,
		marginBottom:      marginBottom,
		marginLeft:        marginLeft,
		marginRight:       marginRight,
		marginUnit:        marginUnit,
		textAlign:         textAlign,
		rawHTMLTag:        rawHTMLTag,
		enableRawHTML:     enableRawHTML,
		tocSpace:          tocSpace,
		flowSize:          flowSize,
		emptyChapterStyle: emptyChapterStyle,
		coverStyle:        coverStyle,
		titleFont:         titleFont,
		authorFont:        authorFont,
		screenHeight:      screenHeight,
		fontType:          fontType,
	})

	if outPath == "" {
		outPath = defaultOutputPath(inputPath, ".epub", cfg)
	}
	if !strings.EqualFold(filepath.Ext(outPath), ".epub") {
		return errors.New(lang.ErrOutputMustBeEPUB)
	}

	text, detected, err := textio.DecodeFile(inputPath, encodingName)
	if err != nil {
		return err
	}

	if title == "" {
		title = strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	}
	if author == "" {
		title, author = splitTitleAuthor(title)
	}
	author = normalizeCommaListText(author)
	if description == "" {
		description = cfg.Advanced.Description
	}
	if publisher == "" {
		publisher = cfg.Advanced.Publisher
	}
	if date == "" {
		date = cfg.Advanced.Date
	}
	if identifier == "" {
		identifier = cfg.Advanced.Identifier
	}
	if identifier == "" && isbn != "" {
		identifier = "urn:isbn:" + isbn
	}
	if rights == "" {
		rights = cfg.Advanced.Rights
	}
	subjects = append(subjects, category...)
	fontTypeFlag := hasFlag(args, "font-type")
	if fontTypeFlag && !cfg.Recent.UseEmbeddedFontsBool() {
		fonts = nil
	} else if len(fonts) == 0 {
		fonts = append(fonts, cfg.Recent.EmbeddedFonts()...)
	}
	useFontSources := []string(fontSources)
	if len(useFontSources) == 0 {
		useFontSources, err = readerFontSources(cfg, cfgPath)
		if err != nil {
			return err
		}
	}
	if fontTypeFlag && !cfg.Recent.UseCustomizedFontsBool() && !cfg.Recent.UseMachineFontsBool() {
		useFontSources = nil
	}
	if fontTypeFlag && !cfg.Recent.UseEmbeddedFontsBool() {
		subsetFonts = false
	} else if !hasFlag(args, "subset-fonts") {
		subsetFonts = cfg.Recent.UseEmbeddedFontsBool() && cfg.Recent.FontSubsettingBool()
	}
	if splitCount == 0 && !splitCountFlag {
		splitCount = cfg.SplitCount()
	}

	customCSS, err := readOptional(cssPath)
	if err != nil {
		return err
	}
	if cssPath == "" {
		customCSS, err = loadCachedCSS(cfg.Recent.SaveCSSBool())
		if err != nil {
			return err
		}
	} else if err := saveCachedCSS(cfg.Recent.SaveCSSBool(), customCSS); err != nil {
		return err
	}
	if vertical {
		customCSS += "\n" + epub.VerticalCSS
	}
	cssMode = cfg.Recent.CSSOverwriteInt()
	if cssPath != "" && cssMode == 0 {
		cssMode = 1
	}

	chapterRegex := regex
	if chapterRegex == "" {
		chapterRegex = cfg.ChapterRegex()
	}

	var chapterPlan []book.ChapterMark
	if chapterFile != "" {
		chapterPlan, err = book.LoadChapterPlan(chapterFile)
		if err != nil {
			return err
		}
	}

	imageResolver, scannedImages, err := scanImageDirs(scanImages)
	if err != nil {
		return err
	}
	if imageList != "" {
		listed, err := loadImageManifest(imageList)
		if err != nil {
			return err
		}
		images = append(images, listed...)
		for _, image := range listed {
			imageResolver[strings.ToLower(filepath.Base(image))] = image
		}
	}
	images = append(images, scannedImages...)

	if subsetFonts {
		subsetted, cleanup, err := convert.SubsetFonts(fonts, text, subsetTool)
		if err != nil {
			return err
		}
		defer cleanup()
		fonts = subsetted
	}

	bk, err := book.FromText(text, book.TextOptions{
		Title:               title,
		Author:              author,
		Translator:          translator,
		Language:            language,
		Description:         description,
		Publisher:           publisher,
		Subjects:            subjects,
		Date:                date,
		Identifier:          identifier,
		Rights:              rights,
		ChapterRegex:        chapterRegex,
		ChapterPlan:         chapterPlan,
		FixedChapterLen:     splitLength,
		SplitCount:          splitCount,
		RemoveBlankLine:     cfg.Recent.RemoveBlankLineBool(),
		AddSpace:            cfg.Recent.AddSpaceBool(),
		AddSpaceCount:       cfg.Recent.AddSpaceCountInt(),
		SkipEmptyChapters:   cfg.Advanced.SkipEmptyChaptersBool(),
		EmptyAsSubdirectory: cfg.Advanced.EmptyChaptersAsSubdirectoryBool(),
		RawHTMLTag:          cfg.Advanced.RawHTMLTag(),
		EnableRawHTML:       cfg.Advanced.EnableHTMLRawTagBool(),
		TOCSpace:            cfg.Advanced.TOCSpaceBool(),
		Layout:              cfg.LayoutOptions(),
		CustomCSS:           customCSS,
		CSSMode:             cssMode,
		CoverPath:           coverPath,
		Cover:               cfg.CoverOptions(),
		SkipCoverPage:       coverPath == "" && !cfg.Recent.ForceTextCoverBool(),
		FlowSizeKB:          cfg.Advanced.FlowSizeKB(),
		Fonts:               fonts,
		FontSources:         useFontSources,
		Images:              images,
		ImageResolver:       imageResolver,
	})
	if err != nil {
		return err
	}

	if err := epub.Write(outPath, bk); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, lang.MsgBuilt, outPath, len(bk.Chapters), detected)
	return nil
}

func runChapters(args []string) error {
	fs := flag.NewFlagSet("chapters", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		outPath      string
		configPath   string
		title        string
		encodingName string
		regex        string
		splitLength  int
		splitCount   int
	)
	fs.StringVar(&outPath, "o", "", "chapter TSV output path")
	fs.StringVar(&configPath, "config", "", "EasyPub config.xml path")
	fs.StringVar(&title, "title", "", "book title")
	fs.StringVar(&encodingName, "encoding", "auto", "text encoding: auto, utf-8, gbk, gb18030, utf-16le, utf-16be")
	fs.StringVar(&regex, "chapter-regex", "", "chapter title regular expression")
	fs.IntVar(&splitLength, "split-length", 0, "split chapters by roughly n characters")
	fs.IntVar(&splitCount, "split-count", 0, "split into n even chapters, matching EasyPub length split")
	if err := fs.Parse(interspersedFlags(args, nil)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New(lang.ErrChaptersInputCount)
	}
	splitCountFlag := hasFlag(args, "split-count")

	inputPath := fs.Arg(0)
	cfg, _, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	if outPath == "" {
		outPath = replaceExt(inputPath, ".chapters.tsv")
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	}
	text, detected, err := textio.DecodeFile(inputPath, encodingName)
	if err != nil {
		return err
	}
	chapterRegex := regex
	if chapterRegex == "" {
		chapterRegex = cfg.ChapterRegex()
	}
	if splitCount == 0 && !splitCountFlag {
		splitCount = cfg.SplitCount()
	}
	marks, err := book.DetectChapterPlan(text, book.TextOptions{
		Title:           title,
		ChapterRegex:    chapterRegex,
		FixedChapterLen: splitLength,
		SplitCount:      splitCount,
		TOCSpace:        cfg.Advanced.TOCSpaceBool(),
	})
	if err != nil {
		return err
	}
	if err := book.WriteChapterPlan(outPath, marks); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, lang.MsgChaptersWritten, outPath, len(marks), detected)
	return nil
}

func runTXT(args []string) error {
	fs := flag.NewFlagSet("txt", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var outPath string
	fs.StringVar(&outPath, "o", "", "output txt path")
	if err := fs.Parse(interspersedFlags(args, nil)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New(lang.ErrTXTInputCount)
	}
	inputPath := fs.Arg(0)
	if outPath == "" {
		outPath = replaceExt(inputPath, ".txt")
	}
	text, err := epub.ExtractText(inputPath)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(text), 0644)
}

func runInspectConfig(args []string) error {
	fs := flag.NewFlagSet("inspect-config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New(lang.ErrInspectConfigCount)
	}
	cfg, err := config.Load(fs.Arg(0))
	if err != nil {
		return err
	}
	fmt.Println("chapter regex:", cfg.ChapterRegex())
	fmt.Println("remove blank lines:", cfg.Recent.RemoveBlankLineBool())
	fmt.Println("add full-width spaces:", cfg.Recent.AddSpaceBool())
	fmt.Println("full-width space count:", cfg.Recent.AddSpaceCountInt())
	fmt.Println("split count:", cfg.SplitCount())
	fmt.Println("embedded fonts:", strings.Join(cfg.Recent.EmbeddedFonts(), ", "))
	fmt.Println("reader font sources:", strings.Join(mustReaderFontSources(cfg, fs.Arg(0)), ", "))
	fmt.Println("font subsetting:", cfg.Recent.FontSubsettingBool())
	fmt.Println("force text cover:", cfg.Recent.ForceTextCoverBool())
	fmt.Println("output folder:", cfg.Recent.OutputFolder)
	fmt.Println("output to source:", cfg.Advanced.OutputToSrcBool())
	fmt.Println("raw html tag:", cfg.Advanced.RawHTMLTag())
	fmt.Println("force empty chapters:", cfg.Advanced.ForceEmptyChapterBool())
	fmt.Println("empty chapter style:", cfg.Advanced.EmptyChapterStyleInt())
	fmt.Println("flow size KB:", cfg.Advanced.FlowSizeKB())
	return nil
}

func usage() {
	fmt.Fprint(os.Stderr, lang.Usage)
}

type buildOverrides struct {
	forceTextCover    bool
	removeBlankLine   bool
	addSpace          bool
	addSpaceCount     int
	saveCSS           bool
	cssMode           int
	lineHeight        int
	fontSize          int
	indent            float64
	marginTop         float64
	marginBottom      float64
	marginLeft        float64
	marginRight       float64
	marginUnit        int
	textAlign         int
	rawHTMLTag        string
	enableRawHTML     bool
	tocSpace          bool
	flowSize          int
	emptyChapterStyle int
	coverStyle        int
	titleFont         int
	authorFont        int
	screenHeight      int
	fontType          int
}

func applyBuildOverrides(cfg *config.Config, args []string, ov buildOverrides) {
	if cfg == nil {
		return
	}
	setBool := func(flagName string, value bool, target *string) {
		if hasFlag(args, flagName) {
			*target = boolIntString(value)
		}
	}
	setPositiveInt := func(flagName string, value int, target *string) {
		if hasFlag(args, flagName) && value > 0 {
			*target = strconv.Itoa(value)
		}
	}
	setInt := func(flagName string, value int, target *string) {
		if hasFlag(args, flagName) && value >= 0 {
			*target = strconv.Itoa(value)
		}
	}

	setBool("force-text-cover", ov.forceTextCover, &cfg.Recent.ForceTextCover)
	setBool("remove-blank-line", ov.removeBlankLine, &cfg.Recent.RemoveBlankLine)
	setBool("add-space", ov.addSpace, &cfg.Recent.AddSpace)
	setPositiveInt("add-space-count", ov.addSpaceCount, &cfg.Recent.AddSpaceCount)
	setBool("save-css", ov.saveCSS, &cfg.Recent.SaveCSS)
	setInt("css-mode", ov.cssMode, &cfg.Recent.CSSOverwrite)
	setPositiveInt("line-height", ov.lineHeight, &cfg.Recent.LineHeight)
	setPositiveInt("font-size", ov.fontSize, &cfg.Recent.FontSize)
	if hasFlag(args, "indent") && ov.indent >= 0 {
		cfg.Recent.Indent = strconv.FormatFloat(ov.indent, 'f', -1, 64)
	}
	if hasFlag(args, "margin-top") && ov.marginTop >= 0 {
		cfg.Recent.Top = strconv.FormatFloat(ov.marginTop, 'f', -1, 64)
	}
	if hasFlag(args, "margin-bottom") && ov.marginBottom >= 0 {
		cfg.Recent.Bottom = strconv.FormatFloat(ov.marginBottom, 'f', -1, 64)
	}
	if hasFlag(args, "margin-left") && ov.marginLeft >= 0 {
		cfg.Recent.Left = strconv.FormatFloat(ov.marginLeft, 'f', -1, 64)
	}
	if hasFlag(args, "margin-right") && ov.marginRight >= 0 {
		cfg.Recent.Right = strconv.FormatFloat(ov.marginRight, 'f', -1, 64)
	}
	if hasFlag(args, "margin-unit") && ov.marginUnit >= 0 {
		unit := strconv.Itoa(ov.marginUnit)
		cfg.Recent.PageTopUnit = unit
		cfg.Recent.PageBottomUnit = unit
		cfg.Recent.PageLeftUnit = unit
		cfg.Recent.PageRightUnit = unit
	}
	setInt("text-align", ov.textAlign, &cfg.Recent.TextAlign)
	if hasFlag(args, "raw-html-tag") && strings.TrimSpace(ov.rawHTMLTag) != "" {
		cfg.Advanced.HTMLRawTag = ov.rawHTMLTag
	}
	setBool("enable-raw-html", ov.enableRawHTML, &cfg.Advanced.EnableHTMLRaw)
	setBool("toc-space", ov.tocSpace, &cfg.Advanced.TOCSpace)
	setInt("flow-size", ov.flowSize, &cfg.Advanced.FlowSize)
	setInt("empty-chapter-style", ov.emptyChapterStyle, &cfg.Advanced.EmptyStyle)
	setInt("cover-style", ov.coverStyle, &cfg.Recent.CoverStyle)
	setPositiveInt("cover-title-font", ov.titleFont, &cfg.Recent.TitleFont)
	setPositiveInt("cover-author-font", ov.authorFont, &cfg.Recent.AuthorFont)
	setPositiveInt("screen-height", ov.screenHeight, &cfg.Advanced.ScreenHeight)
	setInt("font-type", ov.fontType, &cfg.Recent.FontType)
}

func boolIntString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func loadConfig(path string) (*config.Config, string, error) {
	if path != "" {
		cfg, err := config.Load(path)
		return cfg, path, err
	}
	for _, candidate := range []string{"config.xml", `C:\PortableApps\easypub\config.xml`} {
		if _, err := os.Stat(candidate); err == nil {
			cfg, err := config.Load(candidate)
			return cfg, candidate, err
		}
	}
	return config.Default(), "", nil
}

func readerFontSources(cfg *config.Config, cfgPath string) ([]string, error) {
	if cfg == nil || cfg.Recent.UseEmbeddedFontsBool() || cfg.Recent.UseDefaultReaderFontsBool() {
		return nil, nil
	}
	if cfg.Recent.UseCustomizedFontsBool() {
		customized := cfg.Recent.CustomizedFonts()
		return customized, nil
	}
	if !cfg.Recent.UseMachineFontsBool() {
		return nil, nil
	}
	if cfgPath == "" || strings.TrimSpace(cfg.EReadersConfig) == "" {
		return nil, nil
	}
	ereadersPath := filepath.Join(filepath.Dir(cfgPath), cfg.EReadersConfig)
	if _, err := os.Stat(ereadersPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	ereaders, err := config.LoadEReaders(ereadersPath)
	if err != nil {
		return nil, err
	}
	model, ok := ereaders.Model(cfg.Recent.MachineIDInt())
	if !ok {
		return nil, nil
	}
	return model.Fonts, nil
}

func mustReaderFontSources(cfg *config.Config, cfgPath string) []string {
	sources, err := readerFontSources(cfg, cfgPath)
	if err != nil {
		return nil
	}
	return sources
}

func readOptional(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func replaceExt(path, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}

func splitTitleAuthor(name string) (string, string) {
	name = strings.TrimSpace(name)
	for _, pair := range []struct{ open, close string }{
		{"[", "]"},
		{"【", "】"},
		{"(", ")"},
		{"（", "）"},
	} {
		if strings.HasSuffix(name, pair.close) {
			if idx := strings.LastIndex(name, pair.open); idx > 0 {
				title := strings.TrimSpace(name[:idx])
				author := strings.TrimSpace(strings.TrimSuffix(name[idx+len(pair.open):], pair.close))
				if title != "" && author != "" {
					return title, author
				}
			}
		}
	}
	for _, sep := range []string{" - ", "－", "—", "–", "_", "．", " . "} {
		if before, after, ok := strings.Cut(name, sep); ok {
			before = strings.TrimSpace(before)
			after = strings.TrimSpace(after)
			if before != "" && after != "" {
				return before, after
			}
		}
	}
	return name, ""
}

func defaultOutputPath(inputPath, ext string, cfg *config.Config) string {
	name := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)) + ext
	if cfg != nil && !cfg.Advanced.OutputToSrcBool() {
		dir := strings.TrimSpace(cfg.Recent.OutputFolder)
		if dir != "" {
			return filepath.Join(dir, name)
		}
	}
	return filepath.Join(filepath.Dir(inputPath), name)
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	return errA == nil && errB == nil && strings.EqualFold(aa, bb)
}

func hasFlag(args []string, name string) bool {
	want := "-" + name
	long := "--" + name
	for _, arg := range args {
		if arg == want || arg == long || strings.HasPrefix(arg, want+"=") || strings.HasPrefix(arg, long+"=") {
			return true
		}
	}
	return false
}

func interspersedFlags(args []string, boolFlags map[string]bool) []string {
	var flags []string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}

		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if cut, _, ok := strings.Cut(name, "="); ok {
			name = cut
		}
		if strings.Contains(arg, "=") || boolFlags[name] {
			continue
		}
		if i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return append(flags, positional...)
}
