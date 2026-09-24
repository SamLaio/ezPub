//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ezpub/internal/book"
	"ezpub/internal/config"
	"ezpub/internal/convert"
	"ezpub/internal/lang"
	"ezpub/internal/textio"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows"
)

var (
	user32                        = windows.NewLazySystemDLL("user32.dll")
	gdi32                         = windows.NewLazySystemDLL("gdi32.dll")
	setProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	setProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	getDC                         = user32.NewProc("GetDC")
	releaseDC                     = user32.NewProc("ReleaseDC")
	getDeviceCaps                 = gdi32.NewProc("GetDeviceCaps")
	guiDPIOnce                    sync.Once
	guiDPIValue                   int
)

const (
	guiWindowWidth   = 845
	guiWindowHeight  = 350
	guiCanvasWidth   = 820
	guiCanvasHeight  = 315
	guiTabHeight     = 260
	guiPageHeight    = 285
	guiBottomRowY    = 282
	guiStatusY       = 286
	guiStartButtonX  = 700
	guiStartButtonW  = 110
	guiStartButtonH  = 28
	guiStatusPadding = 10
)

func initGUIProcessDPI() {
	// Per-monitor v2 is represented by the pseudo handle -4.
	if r, _, _ := setProcessDpiAwarenessContext.Call(^uintptr(3)); r != 0 {
		return
	}
	_, _, _ = setProcessDPIAware.Call()
}

func currentGUIDPI() int {
	guiDPIOnce.Do(func() {
		guiDPIValue = 96
		hdc, _, _ := getDC.Call(0)
		if hdc == 0 {
			return
		}
		defer releaseDC.Call(0, hdc)
		const logPixelsY = 90
		if dpi, _, _ := getDeviceCaps.Call(hdc, logPixelsY); dpi > 0 {
			guiDPIValue = int(dpi)
		}
	})
	return guiDPIValue
}

func ui(value int) int {
	if value == 0 {
		return 0
	}
	dpi := currentGUIDPI()
	return (value*96 + dpi/2) / dpi
}

func uiSize(width, height int) Size {
	return Size{Width: ui(width), Height: ui(height)}
}

func uiW(width int) Size {
	return Size{Width: ui(width)}
}

func uiH(height int) Size {
	return Size{Height: ui(height)}
}

type fixedComposite struct {
	MinSize       Size
	MaxSize       Size
	StretchFactor int
	Row           int
	Column        int
	RowSpan       int
	ColumnSpan    int
	Alignment     Alignment2D
	AssignTo      **walk.Composite
	Children      []Widget
	DataBinder    DataBinder
	Place         func()
}

func (c fixedComposite) Create(builder *Builder) error {
	w, err := walk.NewComposite(builder.Parent())
	if err != nil {
		return err
	}
	if err := setPassiveLayout(w, c.MinSize); err != nil {
		return err
	}
	if c.AssignTo != nil {
		*c.AssignTo = w
	}
	w.SetSuspended(true)
	builder.Defer(func() error {
		w.SetSuspended(false)
		return nil
	})
	return builder.InitWidget(c, w, func() error {
		place := func() {
			_ = w.BringToTop()
			if c.Place != nil {
				c.Place()
			}
			bringChildrenToTop(w)
		}
		builder.Defer(func() error {
			place()
			time.AfterFunc(50*time.Millisecond, func() {
				w.Synchronize(place)
			})
			return nil
		})
		w.SizeChanged().Attach(place)
		w.BoundsChanged().Attach(place)
		return nil
	})
}

func fixedPage(title string, children []Widget, place func()) TabPage {
	pageChildren := append(append([]Widget(nil), children...), placeHook{place: place})
	return TabPage{
		Title:           title,
		Layout:          absoluteDeclLayout{minSize: Size{Width: guiCanvasWidth, Height: guiPageHeight}},
		Children:        pageChildren,
		OnBoundsChanged: place,
		OnSizeChanged:   place,
	}
}

type absoluteDeclLayout struct {
	minSize Size
}

func (l absoluteDeclLayout) Create() (walk.Layout, error) {
	return newAbsoluteLayout(l.minSize), nil
}

type placeHook struct {
	place func()
}

func (h placeHook) Create(builder *Builder) error {
	parent := builder.Parent()
	place := func() {
		if h.place != nil {
			h.place()
		}
		bringChildrenToTop(parent)
	}
	builder.Defer(func() error {
		place()
		return nil
	})
	time.AfterFunc(50*time.Millisecond, func() {
		if parent != nil {
			parent.Synchronize(place)
		}
	})
	time.AfterFunc(200*time.Millisecond, func() {
		if parent != nil {
			parent.Synchronize(place)
		}
	})
	return nil
}

func setPassiveLayout(container walk.Container, minSize Size) error {
	return container.SetLayout(newAbsoluteLayout(minSize))
}

func setWidgetBounds(w walk.Window, x, y, width, height int) {
	if w == nil {
		return
	}
	_ = w.SetBounds(walk.Rectangle{X: x, Y: y, Width: width, Height: height})
}

func bringChildrenToTop(container walk.Container) {
	if container == nil {
		return
	}
	children := container.Children()
	for i := 0; i < children.Len(); i++ {
		_ = children.At(i).BringToTop()
	}
}

type fixedLabel struct {
	AssignTo  *walk.Window
	Text      string
	TextColor walk.Color
}

func (l fixedLabel) Create(builder *Builder) error {
	w, err := walk.NewTextLabel(builder.Parent())
	if err != nil {
		return err
	}
	if l.AssignTo != nil {
		*l.AssignTo = w
	}
	if l.TextColor != 0 {
		w.SetTextColor(l.TextColor)
	}
	return w.SetText(l.Text)
}

type absoluteLayout struct {
	walk.LayoutBase
	container walk.Container
	minSize   walk.Size
}

func newAbsoluteLayout(minSize Size) *absoluteLayout {
	return &absoluteLayout{
		minSize: walk.Size{Width: minSize.Width, Height: minSize.Height},
	}
}

func (l *absoluteLayout) Container() walk.Container {
	return l.container
}

func (l *absoluteLayout) SetContainer(container walk.Container) {
	l.container = container
}

func (l *absoluteLayout) Margins() walk.Margins {
	return walk.Margins{}
}

func (l *absoluteLayout) SetMargins(walk.Margins) error {
	return nil
}

func (l *absoluteLayout) Spacing() int {
	return 0
}

func (l *absoluteLayout) SetSpacing(int) error {
	return nil
}

func (l *absoluteLayout) CreateLayoutItem(ctx *walk.LayoutContext) walk.ContainerLayoutItem {
	return &absoluteLayoutItem{minSize: l.minSize}
}

type absoluteLayoutItem struct {
	walk.ContainerLayoutItemBase
	minSize walk.Size
}

func (*absoluteLayoutItem) LayoutFlags() walk.LayoutFlags {
	return walk.ShrinkableHorz | walk.GrowableHorz | walk.GreedyHorz | walk.ShrinkableVert | walk.GrowableVert | walk.GreedyVert
}

func (li *absoluteLayoutItem) MinSize() walk.Size {
	if li.Geometry().MinSize.Width > 0 || li.Geometry().MinSize.Height > 0 {
		return li.Geometry().MinSize
	}
	return li.minSize
}

func (li *absoluteLayoutItem) MinSizeForSize(walk.Size) walk.Size {
	return li.MinSize()
}

func (li *absoluteLayoutItem) HeightForWidth(int) int {
	return li.MinSize().Height
}

func (*absoluteLayoutItem) PerformLayout() []walk.LayoutResultItem {
	return nil
}

func runGUI(args []string) error {
	initGUIProcessDPI()
	debugEnabled, guiArgs := parseGUIDebugArgs(args)
	_ = initDebugLog(debugEnabled)
	defer closeDebugLog()
	defer debugRecover("runGUI")
	debugLog("runGUI args=%q debug=%v", guiArgs, debugEnabled)

	cfg, cfgPath, err := loadConfig("")
	if err != nil {
		debugLog("load config failed: %v", err)
		cfg = config.Default()
	}
	debugLog("config path=%q", cfgPath)
	state := newGUIState(cfg, cfgPath)
	if len(guiArgs) > 0 {
		state.input = guiArgs[0]
		state.output = defaultGUIOutputPath(state.input)
		state.title = strings.TrimSuffix(filepath.Base(state.input), filepath.Ext(state.input))
		state.title, state.author = splitTitleAuthor(state.title)
	}
	return showGUI(state)
}

func parseGUIDebugArgs(args []string) (bool, []string) {
	debugEnabled := false
	var out []string
	for _, arg := range args {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "-debug=false", "--debug=false", "/debug=false":
			debugEnabled = false
		case "-debug=true", "--debug=true", "/debug=true", "-debug", "--debug", "/debug":
			debugEnabled = true
		default:
			out = append(out, arg)
		}
	}
	return debugEnabled, out
}

type guiState struct {
	cfg        *config.Config
	configPath string
	input      string
	output     string
	title      string
	author     string
}

func newGUIState(cfg *config.Config, cfgPath string) *guiState {
	return &guiState{cfg: cfg, configPath: cfgPath}
}

func showGUI(state *guiState) error {
	defer debugRecover("showGUI")
	debugLog("showGUI input=%q output=%q title=%q author=%q", state.input, state.output, state.title, state.author)

	var mw *walk.MainWindow
	var inputLE, titleLE, authorLE, outputLE, coverLE, cssPathLE *walk.LineEdit
	var translatorLE, isbnLE, publisherLE, rightsLE *walk.LineEdit
	var dateDE *walk.DateEdit
	var subjectCB, languageCB *walk.ComboBox
	var regexLE, customFontLE, fontLE *walk.LineEdit
	var splitCountNE *walk.NumberEdit
	var lineHeightNE, fontSizeNE, indentNE, flowSizeNE *walk.NumberEdit
	var coverTitleFontNE, coverAuthorFontNE, screenWidthNE, screenHeightNE *walk.NumberEdit
	var marginTopNE, marginBottomNE, marginLeftNE, marginRightNE, paragraphSpacingNE *walk.NumberEdit
	var rawTagLE *walk.LineEdit
	var descriptionTE, cssTE *walk.TextEdit
	var imageRefLE *walk.LineEdit
	var statusLabel *walk.Label
	var startBuildBtn *walk.PushButton
	var coverPreview, imagePreview *walk.ImageView
	var mainTabs *walk.TabWidget
	var forceTextCoverCB, verticalCB, subsetFontsCB, removeBlankLineCB *walk.CheckBox
	var enableRawHTMLCB, tocSpaceCB, lineStartSpaceCB, saveCSSCB *walk.CheckBox
	var simpleRB, regexRB, splitCountRB, fileRB *walk.RadioButton
	var customFontRB, embeddedFontRB, defaultFontRB *walk.RadioButton
	var cssSkipRB, cssAppendRB, cssOverwriteRB *walk.RadioButton
	var emptyStyleCB, textAlignCB, coverStyleCB, marginUnitCB *walk.ComboBox
	var simplePrefixCB, simpleNumberCB, simpleUnitCB, simpleExtraCB *walk.ComboBox
	var imageList *walk.ListBox

	guiPrefs, guiPrefsOK := loadGUISettings()
	imageInputs := []string{}
	editorPath := readSettingText("editor.txt")
	if editorPath == "" {
		editorPath = strings.TrimSpace(state.cfg.Recent.Editor)
	}
	fontType := state.cfg.Recent.FontTypeInt()
	if guiPrefsOK && validGUIFontType(guiPrefs.FontType) {
		fontType = guiPrefs.FontType
	}
	if fontType == 0 {
		fontType = 3
	}
	_, subsetToolAvailable := convert.FindFontSubsetter("")
	cssMode := state.cfg.Recent.CSSOverwriteInt()
	if guiPrefsOK && validGUICSSMode(guiPrefs.CSSMode) {
		cssMode = guiPrefs.CSSMode
	}
	var chapterPlanPath string
	var chapterPlanTemp bool
	lastDefaultOutput := strings.TrimSpace(state.output)
	outputTouched := false
	updatingOutput := false
	chapterMode := 1
	setChapterMode := func(mode int) {
		chapterMode = mode
		if simpleRB != nil {
			simpleRB.SetChecked(mode == 0)
		}
		if regexRB != nil {
			regexRB.SetChecked(mode == 1)
		}
		if splitCountRB != nil {
			splitCountRB.SetChecked(mode == 2)
		}
		if fileRB != nil {
			fileRB.SetChecked(mode == 3)
		}
		debugLog("chapter mode=%d", mode)
	}
	defer func() {
		if chapterPlanTemp && chapterPlanPath != "" {
			debugLog("cleanup chapter plan: %q", chapterPlanPath)
			_ = os.Remove(chapterPlanPath)
		}
	}()
	refreshImageSelection := func() {
		path := ""
		if imageList != nil {
			idx := imageList.CurrentIndex()
			if idx >= 0 && idx < len(imageInputs) {
				path = imageInputs[idx]
			}
		}
		if imageRefLE != nil {
			if path != "" && !isDirPath(path) {
				_ = imageRefLE.SetText(imageReference(filepath.Base(path)))
			} else {
				_ = imageRefLE.SetText("")
			}
		}
		setImagePreview(imagePreview, path)
	}
	updateImages := func() {
		oldIndex := -1
		if imageList != nil {
			oldIndex = imageList.CurrentIndex()
			_ = imageList.SetModel(append([]string(nil), imageInputs...))
			if len(imageInputs) > 0 {
				if oldIndex < 0 || oldIndex >= len(imageInputs) {
					oldIndex = 0
				}
				_ = imageList.SetCurrentIndex(oldIndex)
			}
		}
		refreshImageSelection()
	}
	selectedImage := func() string {
		if imageList == nil {
			return ""
		}
		idx := imageList.CurrentIndex()
		if idx < 0 || idx >= len(imageInputs) {
			return ""
		}
		return imageInputs[idx]
	}
	showError := func(err error) {
		if err != nil {
			debugLog("error: %v", err)
			walk.MsgBox(mw, "ezPub", err.Error(), walk.MsgBoxIconError)
			if statusLabel != nil {
				_ = statusLabel.SetText(err.Error())
			}
		}
	}
	splitCountText := func() string {
		if splitCountNE == nil {
			return intText(state.cfg.SplitCount())
		}
		value := int(splitCountNE.Value())
		if value < 1 {
			value = 1
		}
		return strconv.Itoa(value)
	}
	numberIntText := func(ne *walk.NumberEdit, fallback int) string {
		if ne == nil {
			return strconv.Itoa(fallback)
		}
		return strconv.Itoa(int(ne.Value()))
	}
	numberFloatText := func(ne *walk.NumberEdit, fallback float64) string {
		if ne == nil {
			return strconv.FormatFloat(fallback, 'f', -1, 64)
		}
		return strconv.FormatFloat(ne.Value(), 'f', -1, 64)
	}
	setOutputFromInput := func(input string) {
		if strings.TrimSpace(input) == "" {
			return
		}
		if titleLE != nil {
			title, author := splitTitleAuthor(strings.TrimSuffix(filepath.Base(input), filepath.Ext(input)))
			_ = titleLE.SetText(title)
			if authorLE != nil {
				_ = authorLE.SetText(author)
			}
		}
		if outputLE != nil {
			nextOutput := defaultGUIOutputPath(input)
			currentOutput := strings.TrimSpace(outputLE.Text())
			if currentOutput == "" || !outputTouched || samePath(currentOutput, lastDefaultOutput) {
				updatingOutput = true
				_ = outputLE.SetText(nextOutput)
				updatingOutput = false
				lastDefaultOutput = nextOutput
				outputTouched = false
			}
		}
	}
	setCoverPreview := func(path string) {
		setImagePreview(coverPreview, path)
	}

	runBuildFromGUI := func() {
		defer debugRecover("runBuildFromGUI")
		input := strings.TrimSpace(inputLE.Text())
		debugLog("action=start_build input=%q output=%q title=%q author=%q", input, outputLE.Text(), titleLE.Text(), authorLE.Text())
		if input == "" {
			showError(fmt.Errorf("請選擇輸入 TXT"))
			return
		}
		var buildArgs []string
		add := func(name, value string) {
			value = strings.TrimSpace(value)
			if value != "" {
				buildArgs = append(buildArgs, name, value)
			}
		}
		addBool := func(name string, checked bool) {
			buildArgs = append(buildArgs, name+"="+strconv.FormatBool(checked))
		}
		if strings.TrimSpace(outputLE.Text()) != "" {
			add("-o", outputLE.Text())
		}
		authorText := normalizeCommaListText(authorLE.Text())
		_ = authorLE.SetText(authorText)
		add("-config", state.configPath)
		add("-title", titleLE.Text())
		add("-author", authorText)
		add("-translator", translatorLE.Text())
		add("-isbn", isbnLE.Text())
		add("-publisher", publisherLE.Text())
		add("-date", dateEditText(dateDE))
		add("-lang", languageCB.Text())
		add("-description", descriptionTE.Text())
		add("-rights", rightsLE.Text())
		add("-cover", coverLE.Text())
		switch chapterMode {
		case 0:
			add("-chapter-regex", simpleChapterRegexFromUI(simplePrefixCB.Text(), simpleNumberCB.Text(), simpleUnitCB.Text(), simpleExtraCB.Text(), lineStartSpaceCB.Checked()))
		case 1:
			add("-chapter-regex", regexLE.Text())
		case 2:
			add("-split-count", splitCountText())
		case 3:
			if strings.TrimSpace(chapterPlanPath) == "" {
				showError(fmt.Errorf("請先載入或保存章節表"))
				return
			}
			add("-chapters", chapterPlanPath)
		}
		add("-font-type", strconv.Itoa(fontType))
		if fontType == 1 {
			add("-font-source", customFontLE.Text())
		} else if fontType == 2 {
			add("-font", fontLE.Text())
		}
		add("-line-height", numberIntText(lineHeightNE, intValue(state.cfg.Recent.LineHeight, 130)))
		add("-font-size", numberIntText(fontSizeNE, intValue(state.cfg.Recent.FontSize, 100)))
		add("-indent", numberFloatText(indentNE, floatValue(state.cfg.Recent.Indent, 2)))
		add("-margin-top", numberFloatText(marginTopNE, floatValue(state.cfg.Recent.Top, 0)))
		add("-margin-bottom", numberFloatText(marginBottomNE, floatValue(state.cfg.Recent.Bottom, 0)))
		add("-margin-left", numberFloatText(marginLeftNE, floatValue(state.cfg.Recent.Left, 0)))
		add("-margin-right", numberFloatText(marginRightNE, floatValue(state.cfg.Recent.Right, 0)))
		if marginUnitCB != nil && marginUnitCB.CurrentIndex() >= 0 {
			add("-margin-unit", strconv.Itoa(marginUnitCB.CurrentIndex()))
		}
		add("-flow-size", numberIntText(flowSizeNE, state.cfg.Advanced.FlowSizeKB()))
		add("-raw-html-tag", rawTagLE.Text())
		add("-cover-title-font", numberIntText(coverTitleFontNE, intValue(state.cfg.Recent.TitleFont, 50)))
		add("-cover-author-font", numberIntText(coverAuthorFontNE, intValue(state.cfg.Recent.AuthorFont, 25)))
		add("-screen-height", numberIntText(screenHeightNE, intValue(state.cfg.Advanced.ScreenHeight, 720)))
		subjectTags, subjectArgs := guiSubjectBuildArgs(subjectCB.Text())
		_ = subjectCB.SetText(strings.Join(subjectTags, ", "))
		buildArgs = append(buildArgs, subjectArgs...)
		if forceTextCoverCB != nil {
			addBool("-force-text-cover", forceTextCoverCB.Checked())
		}
		if removeBlankLineCB != nil {
			addBool("-remove-blank-line", removeBlankLineCB.Checked())
		}
		addBool("-add-space", false)
		addBool("-save-css", saveCSSCB.Checked())
		addBool("-enable-raw-html", enableRawHTMLCB.Checked())
		addBool("-toc-space", tocSpaceCB.Checked())
		if verticalCB.Checked() {
			buildArgs = append(buildArgs, "-vertical")
		}
		if fontType == 2 && subsetFontsCB != nil && subsetFontsCB.Checked() {
			buildArgs = append(buildArgs, "-subset-fonts")
		}
		buildArgs = append(buildArgs, "-css-mode", strconv.Itoa(cssMode))
		if emptyStyleCB.CurrentIndex() >= 0 {
			buildArgs = append(buildArgs, "-empty-chapter-style", strconv.Itoa(emptyStyleCB.CurrentIndex()))
		}
		if textAlignCB.CurrentIndex() >= 0 {
			buildArgs = append(buildArgs, "-text-align", strconv.Itoa(textAlignCB.CurrentIndex()))
		}
		if coverStyleCB != nil && coverStyleCB.CurrentIndex() >= 0 {
			buildArgs = append(buildArgs, "-cover-style", strconv.Itoa(coverStyleCB.CurrentIndex()))
		}
		if css := strings.TrimSpace(cssTE.Text()); css != "" {
			path, cleanup, err := writeTempCSS(css)
			if err != nil {
				showError(err)
				return
			}
			defer cleanup()
			buildArgs = append(buildArgs, "-css", path)
		} else {
			add("-css", cssPathLE.Text())
		}
		for _, image := range imageInputs {
			info, err := os.Stat(image)
			if err != nil {
				showError(err)
				return
			}
			if info.IsDir() {
				buildArgs = append(buildArgs, "-scan-images", image)
			} else {
				buildArgs = append(buildArgs, "-image", image)
			}
		}
		buildArgs = append(buildArgs, input)
		saveCurrentGUISettings := func() {
			subject := strings.Join(subjectTags, ", ")
			language := strings.TrimSpace(languageCB.Text())
			if err := saveGUISettings(guiSettings{
				FontType:     fontType,
				CustomFont:   customFontLE.Text(),
				EmbeddedFont: fontLE.Text(),
				SubsetFonts:  subsetFontsCB != nil && subsetFontsCB.Checked(),
				Subject:      subject,
				SubjectTags:  subjectTags,
				Language:     language,
				CSSMode:      cssMode,
				VerticalCSS:  verticalCB != nil && verticalCB.Checked(),
				SaveCSS:      saveCSSCB != nil && saveCSSCB.Checked(),
			}); err != nil {
				debugLog("save gui settings failed: %v", err)
			}
			if err := saveTagListEntries(subjectTags); err != nil {
				debugLog("save tag list failed: %v", err)
			} else if subjectCB != nil {
				refreshSubjectDropdown(subjectCB, subjectTags)
			}
			if cssTE != nil {
				if err := writeCachedCSS(cssTE.Text()); err != nil {
					debugLog("save css failed: %v", err)
				}
			}
		}
		saveCurrentGUISettings()
		debugLog("build args=%q", buildArgs)
		_ = statusLabel.SetText("轉換中...")
		if err := runBuild(buildArgs); err != nil {
			showError(err)
			return
		}
		saveCurrentGUISettings()
		msg := "完成：" + strings.TrimSpace(outputLE.Text())
		if strings.TrimSpace(outputLE.Text()) == "" {
			msg = "完成"
		}
		_ = statusLabel.SetText(msg)
		debugLog("build success message=%q", msg)
		walk.MsgBox(mw, "ezPub", msg, walk.MsgBoxIconInformation)
	}
	browseCover := func() {
		defer debugRecover("browseCover")
		debugLog("action=browse_cover")
		path, ok, err := openFileDialog(mw, "選擇封面", "Images (*.jpg;*.jpeg;*.png;*.gif;*.webp)|*.jpg;*.jpeg;*.png;*.gif;*.webp|All files (*.*)|*.*")
		if err != nil {
			showError(err)
			return
		}
		if ok {
			debugLog("cover selected: %q", path)
			if coverLE != nil {
				_ = coverLE.SetText(path)
			}
			setCoverPreview(path)
		}
	}
	selectInputFile := func() {
		defer debugRecover("browseInput")
		debugLog("action=browse_input")
		path, ok, err := openFileDialog(mw, "選擇 TXT", "Text files (*.txt)|*.txt|All files (*.*)|*.*")
		if err != nil {
			showError(err)
			return
		}
		if ok {
			debugLog("input selected: %q", path)
			_ = inputLE.SetText(path)
			setOutputFromInput(path)
		}
	}
	selectOutputFile := func() {
		defer debugRecover("browseOutput")
		debugLog("action=browse_output")
		path, ok, err := saveFileDialog(mw, "輸出 EPUB", "EPUB files (*.epub)|*.epub|All files (*.*)|*.*")
		if err != nil {
			showError(err)
			return
		}
		if ok {
			debugLog("output selected: %q", path)
			outputTouched = true
			_ = outputLE.SetText(ensureExt(path, ".epub"))
		}
	}
	selectTextEditor := func() {
		defer debugRecover("selectTextEditor")
		debugLog("action=select_text_editor")
		path, ok, err := openFileDialog(mw, "Select Text Editor...", "Executable files (*.exe)|*.exe|All files (*.*)|*.*")
		if err != nil {
			showError(err)
			return
		}
		if ok {
			editorPath = path
			if err := writeSettingText("editor.txt", editorPath); err != nil {
				showError(err)
				return
			}
			debugLog("text editor selected: %q", editorPath)
			_ = statusLabel.SetText("已選擇 TXT 編輯器：" + editorPath)
		}
	}
	openTXTInEditor := func() {
		input := strings.TrimSpace(inputLE.Text())
		if input == "" {
			showError(fmt.Errorf("請先選擇輸入 TXT"))
			return
		}
		debugLog("action=open_txt input=%q editor=%q", input, editorPath)
		if err := openTextFileWithEditor(input, editorPath); err != nil {
			showError(err)
		}
	}
	handleDropFiles := func(paths []string) {
		defer debugRecover("handleDropFiles")
		debugLog("action=drop_files paths=%q", paths)
		if mainTabs != nil && mainTabs.CurrentIndex() == 6 {
			var added []string
			for _, path := range paths {
				if isImagePath(path) || isDirPath(path) {
					added = append(added, path)
				}
			}
			if len(added) > 0 {
				imageInputs = appendUniquePaths(imageInputs, added...)
				updateImages()
				_ = statusLabel.SetText("成功加入文件：" + strings.Join(added, "；"))
				return
			}
		}
		for _, path := range paths {
			switch {
			case strings.EqualFold(filepath.Ext(path), ".txt"):
				_ = inputLE.SetText(path)
				setOutputFromInput(path)
				_ = statusLabel.SetText("成功載入文件：" + path)
				return
			case isImagePath(path):
				_ = coverLE.SetText(path)
				setCoverPreview(path)
				_ = statusLabel.SetText("成功載入封面：" + path)
				return
			}
		}
		_ = statusLabel.SetText("拖曳檔案未支援")
	}
	loadChapterFile := func() {
		defer debugRecover("loadChapterFile")
		debugLog("action=load_chapter_file")
		path, ok, err := openFileDialog(mw, "載入章節表", "TSV files (*.tsv)|*.tsv|All files (*.*)|*.*")
		if err != nil {
			showError(err)
			return
		}
		if !ok {
			return
		}
		if _, err := book.LoadChapterPlan(path); err != nil {
			showError(err)
			return
		}
		chapterPlanPath = path
		chapterPlanTemp = false
		setChapterMode(3)
		_ = statusLabel.SetText("已載入章節表：" + path)
		debugLog("chapter file loaded: %q", path)
	}

	exportChapters := func(openAfter bool) {
		defer debugRecover("exportChapters")
		input := strings.TrimSpace(inputLE.Text())
		debugLog("action=export_chapters openAfter=%v input=%q", openAfter, input)
		if input == "" {
			showError(fmt.Errorf("請先選擇輸入 TXT"))
			return
		}
		out := chapterPlanPath
		if out == "" {
			file, err := os.CreateTemp("", "ezpub-chapters-*.tsv")
			if err != nil {
				showError(err)
				return
			}
			out = file.Name()
			_ = file.Close()
			chapterPlanPath = out
			chapterPlanTemp = true
		}
		if openAfter {
			editorRegex := regexLE.Text()
			if chapterMode == 0 {
				editorRegex = simpleChapterRegexFromUI(simplePrefixCB.Text(), simpleNumberCB.Text(), simpleUnitCB.Text(), simpleExtraCB.Text(), lineStartSpaceCB.Checked())
			}
			marks, err := detectGUIChapters(input, state.configPath, titleLE.Text(), chapterMode, editorRegex, splitCountText(), chapterPlanPath)
			if err != nil {
				showError(err)
				return
			}
			sourceText, _, err := textio.DecodeFile(input, "auto")
			if err != nil {
				showError(err)
				return
			}
			editorTitle := "當前正則表達式："
			editorStatus := regexLE.Text()
			switch chapterMode {
			case 0:
				editorStatus = editorRegex
			case 2:
				editorTitle = "按長度均分"
				editorStatus = "未使用正則表達式或無法找到對應的內容。按長度分成章"
			case 3:
				editorTitle = "從文件加載"
				editorStatus = chapterPlanPath
			}
			editedMarks, ok := showChapterEditor(mw, editorTitle, editorStatus, out, marks, guiTextLines(sourceText))
			if !ok {
				return
			}
			if err := book.WriteChapterPlan(out, editedMarks); err != nil {
				showError(err)
				return
			}
			setChapterMode(3)
			_ = statusLabel.SetText("章節表已輸出：" + out)
			debugLog("chapters saved from editor: %q", out)
			return
		}
		args := []string{"-o", out}
		if state.configPath != "" {
			args = append(args, "-config", state.configPath)
		}
		if strings.TrimSpace(titleLE.Text()) != "" {
			args = append(args, "-title", titleLE.Text())
		}
		switch chapterMode {
		case 0:
			args = append(args, "-chapter-regex", simpleChapterRegexFromUI(simplePrefixCB.Text(), simpleNumberCB.Text(), simpleUnitCB.Text(), simpleExtraCB.Text(), lineStartSpaceCB.Checked()))
		case 1:
			if strings.TrimSpace(regexLE.Text()) != "" {
				args = append(args, "-chapter-regex", regexLE.Text())
			}
		case 2:
			args = append(args, "-split-count", splitCountText())
		case 3:
			if strings.TrimSpace(chapterPlanPath) != "" {
				args = append(args, "-chapters", chapterPlanPath)
			}
		}
		args = append(args, input)
		if err := runChapters(args); err != nil {
			showError(err)
			return
		}
		_ = statusLabel.SetText("章節表已輸出：" + out)
		debugLog("chapters exported: %q", out)
	}

	window := MainWindow{
		AssignTo:    &mw,
		Title:       "ezPub " + lang.Version + " Sam版",
		Size:        Size{Width: guiWindowWidth, Height: guiWindowHeight},
		MinSize:     Size{Width: guiWindowWidth, Height: guiWindowHeight},
		MaxSize:     Size{Width: guiWindowWidth, Height: guiWindowHeight},
		Layout:      Flow{MarginsZero: true, SpacingZero: true},
		OnDropFiles: handleDropFiles,
		Children: []Widget{
			fixedComposite{
				MinSize: Size{Width: guiCanvasWidth, Height: guiCanvasHeight},
				MaxSize: Size{Width: guiCanvasWidth, Height: guiCanvasHeight},
				Children: []Widget{
					TabWidget{
						AssignTo:      &mainTabs,
						MinSize:       Size{Width: guiCanvasWidth, Height: guiTabHeight},
						MaxSize:       Size{Width: guiCanvasWidth, Height: guiTabHeight},
						StretchFactor: 0,
						OnCurrentIndexChanged: func() {
							debugLog("action=tab_changed")
						},
						Pages: []TabPage{
							basicTab(&inputLE, &titleLE, &authorLE, &outputLE, state, selectInputFile, selectOutputFile, openTXTInEditor, selectTextEditor, runBuildFromGUI, func() { outputTouched = true }, func() bool { return updatingOutput }),
							chapterTab(&simpleRB, &regexRB, &splitCountRB, &fileRB, &lineStartSpaceCB, &simplePrefixCB, &simpleNumberCB, &simpleUnitCB, &simpleExtraCB, &regexLE, &splitCountNE, state.cfg, setChapterMode, loadChapterFile, exportChapters),
							layoutTab(&marginTopNE, &marginBottomNE, &marginLeftNE, &marginRightNE, &marginUnitCB, &lineHeightNE, &fontSizeNE, &paragraphSpacingNE, &indentNE, &textAlignCB, &removeBlankLineCB, state.cfg),
							coverTab(&coverLE, &coverPreview, &forceTextCoverCB, &coverStyleCB, &coverTitleFontNE, &coverAuthorFontNE, state.cfg, browseCover, setCoverPreview),
							fontTab(&customFontRB, &embeddedFontRB, &defaultFontRB, &customFontLE, &fontLE, &subsetFontsCB, state.cfg, &fontType, guiPrefs, guiPrefsOK, subsetToolAvailable, showError, mw),
							metadataTab(&translatorLE, &isbnLE, &publisherLE, &dateDE, &subjectCB, &languageCB, &rightsLE, &descriptionTE, state.cfg, guiPrefs, guiPrefsOK),
							imagesTab(&imageList, &imagePreview, &imageRefLE, &imageInputs, updateImages, refreshImageSelection, selectedImage, showError, mw),
							advancedTab(&rawTagLE, &flowSizeNE, &emptyStyleCB, &enableRawHTMLCB, &tocSpaceCB, &screenWidthNE, &screenHeightNE, state.cfg),
							cssTab(&cssPathLE, &cssTE, &cssSkipRB, &cssAppendRB, &cssOverwriteRB, &verticalCB, &saveCSSCB, state.cfg, &cssMode, guiPrefs, guiPrefsOK, showError, mw),
						},
					},
					Label{AssignTo: &statusLabel, Text: ""},
					PushButton{AssignTo: &startBuildBtn, Text: "開始轉換", OnClicked: runBuildFromGUI},
				},
				Place: func() {
					setWidgetBounds(mainTabs, 0, 0, guiCanvasWidth, guiTabHeight)
					setWidgetBounds(statusLabel, 0, guiStatusY, guiStartButtonX-guiStatusPadding, 24)
					setWidgetBounds(startBuildBtn, guiStartButtonX, guiBottomRowY, guiStartButtonW, guiStartButtonH)
				},
			},
		},
	}
	if err := window.Create(); err != nil {
		return err
	}
	if guiPrefsOK {
		if strings.TrimSpace(guiPrefs.Language) != "" && languageCB != nil {
			_ = languageCB.SetText(strings.TrimSpace(guiPrefs.Language))
		}
	}
	if subjectCB != nil {
		installSubjectAppendBehavior(subjectCB)
	}
	setNumberValue := func(ne *walk.NumberEdit, min, max, value float64) {
		if ne == nil {
			return
		}
		_ = ne.SetRange(min, max)
		if value < min {
			value = min
		}
		if value > max {
			value = max
		}
		_ = ne.SetValue(value)
	}
	if splitCountNE != nil {
		setNumberValue(splitCountNE, 1, 9999, float64(maxInt(1, state.cfg.SplitCount())))
	}
	setNumberValue(marginTopNE, 0, 9999, floatValue(state.cfg.Recent.Top, 0))
	setNumberValue(marginBottomNE, 0, 9999, floatValue(state.cfg.Recent.Bottom, 0))
	setNumberValue(marginLeftNE, 0, 9999, floatValue(state.cfg.Recent.Left, 0))
	setNumberValue(marginRightNE, 0, 9999, floatValue(state.cfg.Recent.Right, 0))
	setNumberValue(fontSizeNE, 1, 9999, float64(intValue(state.cfg.Recent.FontSize, 100)))
	setNumberValue(lineHeightNE, 1, 9999, float64(intValue(state.cfg.Recent.LineHeight, 130)))
	setNumberValue(paragraphSpacingNE, 0, 9999, floatValue(state.cfg.Recent.MarginTop, 1.5))
	setNumberValue(indentNE, 0, 9999, floatValue(state.cfg.Recent.Indent, 2))
	setNumberValue(coverTitleFontNE, 1, 9999, float64(intValue(state.cfg.Recent.TitleFont, 50)))
	setNumberValue(coverAuthorFontNE, 1, 9999, float64(intValue(state.cfg.Recent.AuthorFont, 25)))
	setNumberValue(flowSizeNE, 0, 999999, float64(state.cfg.Advanced.FlowSizeKB()))
	setNumberValue(screenWidthNE, 1, 99999, floatValue(state.cfg.Advanced.ScreenWidth, 540))
	setNumberValue(screenHeightNE, 1, 99999, floatValue(state.cfg.Advanced.ScreenHeight, 720))
	setChapterMode(chapterMode)
	setFontTypeRadio(fontType, customFontRB, embeddedFontRB, defaultFontRB)
	setCSSModeRadio(cssMode, cssSkipRB, cssAppendRB, cssOverwriteRB)
	mw.Show()
	mw.Run()
	return nil
}

func basicTab(inputLE, titleLE, authorLE, outputLE **walk.LineEdit, state *guiState, selectInputFile, selectOutputFile, openTXTInEditor, selectTextEditor, runBuildFromGUI func(), markOutputTouched func(), isUpdatingOutput func() bool) TabPage {
	var inputL, titleL, authorL, outputL walk.Window
	var inputBtn, outputBtn, editBtn, editorBtn *walk.PushButton
	children := []Widget{
		fixedLabel{AssignTo: &inputL, Text: "輸入"},
		LineEdit{AssignTo: inputLE, Text: state.input},
		PushButton{AssignTo: &inputBtn, Text: "...", OnClicked: selectInputFile},
		fixedLabel{AssignTo: &titleL, Text: "書名"},
		LineEdit{AssignTo: titleLE, Text: state.title},
		fixedLabel{AssignTo: &authorL, Text: "作者"},
		LineEdit{AssignTo: authorLE, Text: state.author},
		fixedLabel{AssignTo: &outputL, Text: "輸出"},
		LineEdit{AssignTo: outputLE, Text: state.output, OnTextChanged: func() {
			if !isUpdatingOutput() {
				markOutputTouched()
			}
		}},
		PushButton{AssignTo: &outputBtn, Text: "...", OnClicked: selectOutputFile},
		PushButton{AssignTo: &editBtn, Text: "編輯TXT文件", OnClicked: openTXTInEditor},
		PushButton{AssignTo: &editorBtn, Text: "...", OnClicked: selectTextEditor},
	}
	return fixedPage("基本", children, func() {
		setWidgetBounds(inputL, 12, 18, 42, 24)
		setWidgetBounds(*inputLE, 64, 14, 620, 28)
		setWidgetBounds(inputBtn, 696, 14, 44, 28)
		setWidgetBounds(titleL, 12, 58, 42, 24)
		setWidgetBounds(*titleLE, 64, 54, 270, 28)
		setWidgetBounds(authorL, 350, 58, 42, 24)
		setWidgetBounds(*authorLE, 402, 54, 270, 28)
		setWidgetBounds(outputL, 12, 98, 42, 24)
		setWidgetBounds(*outputLE, 64, 94, 620, 28)
		setWidgetBounds(outputBtn, 696, 94, 44, 28)
		setWidgetBounds(editBtn, 64, 136, 150, 30)
		setWidgetBounds(editorBtn, 226, 136, 44, 30)
	})
}

func chapterTab(simpleRB, regexRB, splitCountRB, fileRB **walk.RadioButton, lineStartSpaceCB **walk.CheckBox, simplePrefixCB, simpleNumberCB, simpleUnitCB, simpleExtraCB **walk.ComboBox, regexLE **walk.LineEdit, splitCountNE **walk.NumberEdit, cfg *config.Config, setChapterMode func(int), loadChapterFile func(), exportChapters func(bool)) TabPage {
	var extraL, splitUnitL walk.Window
	var loadBtn, saveBtn, editBtn *walk.PushButton
	children := []Widget{
		RadioButton{AssignTo: simpleRB, Text: "簡易規則", OnClicked: func() { setChapterMode(0) }},
		CheckBox{AssignTo: lineStartSpaceCB, Text: "行首空格"},
		ComboBox{AssignTo: simplePrefixCB, Editable: true, Model: []string{"第", "卷", "[第卷]"}, CurrentIndex: 0},
		ComboBox{AssignTo: simpleNumberCB, Editable: true, Model: []string{"混合型數字", "純中文數字", "純阿拉伯數字"}, CurrentIndex: 0},
		ComboBox{AssignTo: simpleUnitCB, Editable: true, Model: []string{"章", "回", "卷", "節", "集", "部", "[章回卷節集部]"}, CurrentIndex: 0},
		fixedLabel{AssignTo: &extraL, Text: "附加規則"},
		ComboBox{AssignTo: simpleExtraCB, Editable: true, Model: []string{`^\s*(簡介|序言|序[1-9]|序曲|簡介|後記|尾聲)`}, CurrentIndex: 0},
		RadioButton{AssignTo: regexRB, Text: "正則表達式", OnClicked: func() { setChapterMode(1) }},
		LineEdit{AssignTo: regexLE, Text: cfg.ChapterRegex()},
		RadioButton{AssignTo: splitCountRB, Text: "按長度均分", OnClicked: func() { setChapterMode(2) }},
		NumberEdit{AssignTo: splitCountNE, Decimals: 0, Increment: 1, SpinButtonsVisible: true},
		fixedLabel{AssignTo: &splitUnitL, Text: "章"},
		RadioButton{AssignTo: fileRB, Text: "從文件加載", OnClicked: func() { setChapterMode(3) }},
		PushButton{AssignTo: &loadBtn, Text: "...", OnClicked: loadChapterFile},
		PushButton{AssignTo: &saveBtn, Text: "保存", OnClicked: func() { exportChapters(false) }},
		PushButton{AssignTo: &editBtn, Text: "章節編輯", OnClicked: func() { exportChapters(true) }},
	}
	return fixedPage("章節", children, func() {
		setWidgetBounds(*simpleRB, 18, 18, 90, 26)
		setWidgetBounds(*lineStartSpaceCB, 120, 18, 80, 26)
		setWidgetBounds(*simplePrefixCB, 215, 16, 58, 28)
		setWidgetBounds(*simpleNumberCB, 285, 16, 120, 28)
		setWidgetBounds(*simpleUnitCB, 418, 16, 125, 28)
		setWidgetBounds(extraL, 18, 68, 80, 24)
		setWidgetBounds(*simpleExtraCB, 135, 64, 390, 28)
		setWidgetBounds(*regexRB, 18, 116, 105, 26)
		setWidgetBounds(*regexLE, 135, 112, 390, 28)
		setWidgetBounds(*splitCountRB, 18, 164, 105, 26)
		setWidgetBounds(*splitCountNE, 135, 160, 78, 28)
		setWidgetBounds(splitUnitL, 225, 164, 24, 24)
		setWidgetBounds(*fileRB, 18, 212, 105, 26)
		setWidgetBounds(loadBtn, 135, 208, 44, 30)
		setWidgetBounds(saveBtn, 620, 208, 70, 30)
		setWidgetBounds(editBtn, 705, 208, 90, 30)
	})
}

func layoutTab(marginTopNE, marginBottomNE, marginLeftNE, marginRightNE **walk.NumberEdit, marginUnitCB **walk.ComboBox, lineHeightNE, fontSizeNE, paragraphSpacingNE, indentNE **walk.NumberEdit, textAlignCB **walk.ComboBox, removeBlankLineCB **walk.CheckBox, cfg *config.Config) TabPage {
	num := func(assign **walk.NumberEdit, decimals int, increment float64) NumberEdit {
		return NumberEdit{AssignTo: assign, Decimals: decimals, Increment: increment, SpinButtonsVisible: true}
	}
	marginUnit := intValue(cfg.Recent.PageTopUnit, 0)
	if cfg.Recent.PageTopUnit == "" {
		marginUnit = intValue(cfg.Recent.PageRightUnit, 0)
	}
	var marginL, unitL, topL, bottomL, leftL, rightL, indentL, emL, alignL, fontSizeL, percent1L, lineHeightL, percent2L, removeBlankL walk.Window
	children := []Widget{
		fixedLabel{AssignTo: &marginL, Text: "頁邊距"},
		fixedLabel{AssignTo: &unitL, Text: "單位"},
		ComboBox{AssignTo: marginUnitCB, Model: []string{"px", "%", "em"}, CurrentIndex: marginUnit},
		fixedLabel{AssignTo: &topL, Text: "上"},
		num(marginTopNE, 1, 1),
		fixedLabel{AssignTo: &bottomL, Text: "下"},
		num(marginBottomNE, 1, 1),
		fixedLabel{AssignTo: &leftL, Text: "左"},
		num(marginLeftNE, 1, 1),
		fixedLabel{AssignTo: &rightL, Text: "右"},
		num(marginRightNE, 1, 1),
		fixedLabel{AssignTo: &indentL, Text: "行首縮進"},
		num(indentNE, 1, 0.1),
		fixedLabel{AssignTo: &emL, Text: "em"},
		fixedLabel{AssignTo: &alignL, Text: "對齊"},
		ComboBox{AssignTo: textAlignCB, Model: []string{"默認", "靠左", "置中", "靠右", "左右對齊"}, CurrentIndex: intValue(cfg.Recent.TextAlign, 0)},
		fixedLabel{AssignTo: &fontSizeL, Text: "字體大小"},
		num(fontSizeNE, 0, 1),
		fixedLabel{AssignTo: &percent1L, Text: "%"},
		fixedLabel{AssignTo: &lineHeightL, Text: "行距"},
		num(lineHeightNE, 0, 1),
		fixedLabel{AssignTo: &percent2L, Text: "%"},
		fixedLabel{AssignTo: &removeBlankL, Text: "去除空行"},
		CheckBox{AssignTo: removeBlankLineCB, Checked: cfg.Recent.RemoveBlankLineBool()},
	}
	return fixedPage("版式", children, func() {
		setWidgetBounds(marginL, 18, 18, 80, 24)
		setWidgetBounds(unitL, 245, 18, 40, 24)
		setWidgetBounds(*marginUnitCB, 288, 14, 70, 28)
		setWidgetBounds(topL, 18, 64, 20, 24)
		setWidgetBounds(*marginTopNE, 54, 60, 82, 28)
		setWidgetBounds(bottomL, 155, 64, 20, 24)
		setWidgetBounds(*marginBottomNE, 190, 60, 82, 28)
		setWidgetBounds(leftL, 18, 108, 20, 24)
		setWidgetBounds(*marginLeftNE, 54, 104, 82, 28)
		setWidgetBounds(rightL, 155, 108, 20, 24)
		setWidgetBounds(*marginRightNE, 190, 104, 82, 28)
		setWidgetBounds(indentL, 18, 154, 80, 24)
		setWidgetBounds(*indentNE, 135, 150, 82, 28)
		setWidgetBounds(emL, 228, 154, 30, 24)
		setWidgetBounds(alignL, 18, 200, 80, 24)
		setWidgetBounds(*textAlignCB, 135, 196, 170, 28)
		setWidgetBounds(fontSizeL, 430, 18, 80, 24)
		setWidgetBounds(*fontSizeNE, 540, 14, 82, 28)
		setWidgetBounds(percent1L, 632, 18, 22, 24)
		setWidgetBounds(lineHeightL, 430, 64, 80, 24)
		setWidgetBounds(*lineHeightNE, 540, 60, 82, 28)
		setWidgetBounds(percent2L, 632, 64, 22, 24)
		setWidgetBounds(removeBlankL, 430, 108, 80, 24)
		setWidgetBounds(*removeBlankLineCB, 540, 106, 28, 24)
	})
}

func coverTab(coverLE **walk.LineEdit, coverPreview **walk.ImageView, forceTextCoverCB **walk.CheckBox, coverStyleCB **walk.ComboBox, coverTitleFontNE, coverAuthorFontNE **walk.NumberEdit, cfg *config.Config, browseCover func(), setCoverPreview func(string)) TabPage {
	var coverL, styleL, titleFontL, authorFontL walk.Window
	var browseBtn *walk.PushButton
	children := []Widget{
		fixedLabel{AssignTo: &coverL, Text: "封面圖片"},
		LineEdit{AssignTo: coverLE, OnTextChanged: func() {
			if coverLE != nil && *coverLE != nil {
				setCoverPreview((*coverLE).Text())
			}
		}},
		PushButton{AssignTo: &browseBtn, Text: "...", OnClicked: browseCover},
		CheckBox{AssignTo: forceTextCoverCB, Text: "文字封面", Checked: defaultForceTextCoverChecked("", cfg)},
		fixedLabel{AssignTo: &styleL, Text: "封面樣式"},
		ComboBox{AssignTo: coverStyleCB, Model: []string{"—", "寬度適合", "高度適合"}, CurrentIndex: intValue(cfg.Recent.CoverStyle, intValue(cfg.Advanced.CoverStyle, 0))},
		fixedLabel{AssignTo: &titleFontL, Text: "書名字體"},
		NumberEdit{AssignTo: coverTitleFontNE, Decimals: 0, Increment: 1, SpinButtonsVisible: true},
		fixedLabel{AssignTo: &authorFontL, Text: "作者字體"},
		NumberEdit{AssignTo: coverAuthorFontNE, Decimals: 0, Increment: 1, SpinButtonsVisible: true},
		ImageView{AssignTo: coverPreview, Mode: ImageViewModeZoom, OnMouseDown: func(x, y int, button walk.MouseButton) { browseCover() }},
	}
	return fixedPage("封面", children, func() {
		setWidgetBounds(coverL, 18, 20, 80, 24)
		setWidgetBounds(*coverLE, 110, 16, 330, 28)
		setWidgetBounds(browseBtn, 455, 16, 44, 28)
		setWidgetBounds(*forceTextCoverCB, 110, 58, 100, 26)
		setWidgetBounds(styleL, 18, 102, 80, 24)
		setWidgetBounds(*coverStyleCB, 110, 98, 170, 28)
		setWidgetBounds(titleFontL, 18, 146, 80, 24)
		setWidgetBounds(*coverTitleFontNE, 110, 142, 82, 28)
		setWidgetBounds(authorFontL, 18, 190, 80, 24)
		setWidgetBounds(*coverAuthorFontNE, 110, 186, 82, 28)
		setWidgetBounds(*coverPreview, 585, 22, 150, 150)
	})
}

func fontTab(customFontRB, embeddedFontRB, defaultFontRB **walk.RadioButton, customFontLE, fontLE **walk.LineEdit, subsetFontsCB **walk.CheckBox, cfg *config.Config, fontType *int, prefs guiSettings, prefsOK bool, subsetToolAvailable bool, showError func(error), owner walk.Form) TabPage {
	fonts := cfg.Recent.EmbeddedFonts()
	fontText := ""
	if len(fonts) > 0 {
		fontText = fonts[0]
	}
	if prefsOK {
		fontText = firstNonEmpty(prefs.EmbeddedFont, fontText)
	}
	customFonts := cfg.Recent.CustomizedFonts()
	customFontText := ""
	if len(customFonts) > 0 {
		customFontText = customFonts[0]
	}
	if prefsOK {
		customFontText = firstNonEmpty(prefs.CustomFont, customFontText)
	}
	if fontType == nil {
		defaultFontType := cfg.Recent.FontTypeInt()
		if defaultFontType == 0 {
			defaultFontType = 3
		}
		fontType = &defaultFontType
	}
	selectFontType := func(value int) {
		*fontType = value
		if *customFontRB != nil {
			(*customFontRB).SetChecked(value == 1)
		}
		if *embeddedFontRB != nil {
			(*embeddedFontRB).SetChecked(value == 2)
		}
		if *defaultFontRB != nil {
			(*defaultFontRB).SetChecked(value == 3)
		}
	}
	var browseBtn *walk.PushButton
	children := []Widget{
		RadioButton{AssignTo: customFontRB, Text: "自定義", Value: 1, OnClicked: func() { selectFontType(1) }},
		LineEdit{AssignTo: customFontLE, Text: customFontText},
	}
	embeddedChildren := []Widget{
		RadioButton{AssignTo: embeddedFontRB, Text: "內嵌字體", Value: 2, OnClicked: func() { selectFontType(2) }},
		LineEdit{AssignTo: fontLE, Text: fontText},
		PushButton{AssignTo: &browseBtn, Text: "...", OnClicked: func() {
			defer debugRecover("browseFont")
			debugLog("action=browse_font")
			path, ok, err := openFileDialog(owner, "選擇字體", "Font files (*.ttf;*.otf)|*.ttf;*.otf|All files (*.*)|*.*")
			if err != nil {
				showError(err)
				return
			}
			if ok {
				debugLog("font selected: %q", path)
				selectFontType(2)
				_ = (*fontLE).SetText(path)
			}
		}},
	}
	if subsetToolAvailable {
		subsetChecked := cfg.Recent.FontSubsettingBool()
		if prefsOK {
			subsetChecked = prefs.SubsetFonts
		}
		embeddedChildren = append(embeddedChildren, CheckBox{AssignTo: subsetFontsCB, Text: "僅嵌入子集", Checked: subsetChecked})
	}
	children = append(children, embeddedChildren...)
	children = append(children, RadioButton{AssignTo: defaultFontRB, Text: "使用閱讀器默認字體", Value: 3, OnClicked: func() { selectFontType(3) }})
	return fixedPage("字體", children, func() {
		setWidgetBounds(*customFontRB, 18, 28, 120, 26)
		setWidgetBounds(*customFontLE, 150, 24, 380, 28)
		setWidgetBounds(*embeddedFontRB, 18, 100, 120, 26)
		setWidgetBounds(*fontLE, 150, 96, 270, 28)
		setWidgetBounds(browseBtn, 435, 96, 44, 28)
		if subsetToolAvailable && subsetFontsCB != nil && *subsetFontsCB != nil {
			setWidgetBounds(*subsetFontsCB, 495, 98, 110, 26)
		}
		setWidgetBounds(*defaultFontRB, 18, 172, 190, 26)
	})
}

func setFontTypeRadio(fontType int, customFontRB, embeddedFontRB, defaultFontRB *walk.RadioButton) {
	if customFontRB != nil {
		customFontRB.SetChecked(fontType == 1)
	}
	if embeddedFontRB != nil {
		embeddedFontRB.SetChecked(fontType == 2)
	}
	if defaultFontRB != nil {
		defaultFontRB.SetChecked(fontType == 3)
	}
}

func metadataTab(translatorLE, isbnLE, publisherLE **walk.LineEdit, dateDE **walk.DateEdit, subjectCB, languageCB **walk.ComboBox, rightsLE **walk.LineEdit, descriptionTE **walk.TextEdit, cfg *config.Config, prefs guiSettings, prefsOK bool) TabPage {
	date := parseBookDate(cfg.Advanced.Date)
	language := "zh-CN"
	if prefsOK {
		language = firstNonEmpty(prefs.Language, language)
	}
	subjects := loadTagList()
	if prefsOK {
		subjects = prependUnique(subjects, guiSettingsSubjectTags(prefs)...)
	}
	languages := []string{"zh-CN", "zh-TW", "en-US"}
	languages = prependUnique(languages, language)
	var translatorL, isbnL, dateL, publisherL, subjectL, languageL, descL walk.Window
	children := []Widget{
		fixedLabel{AssignTo: &translatorL, Text: "譯者"},
		LineEdit{AssignTo: translatorLE},
		fixedLabel{AssignTo: &isbnL, Text: "ISBN"},
		LineEdit{AssignTo: isbnLE},
		fixedLabel{AssignTo: &dateL, Text: "出版日期"},
		DateEdit{AssignTo: dateDE, Date: date, Format: "yyyy-MM-dd"},
		fixedLabel{AssignTo: &publisherL, Text: "出版社"},
		LineEdit{AssignTo: publisherLE, Text: cfg.Advanced.Publisher},
		fixedLabel{AssignTo: &subjectL, Text: "標籤"},
		ComboBox{AssignTo: subjectCB, Editable: true, Model: subjects, CurrentIndex: -1},
		fixedLabel{AssignTo: &languageL, Text: "語言"},
		ComboBox{AssignTo: languageCB, Editable: true, Model: languages, CurrentIndex: comboIndex(languages, language)},
		fixedLabel{AssignTo: &descL, Text: "簡介"},
		TextEdit{AssignTo: descriptionTE, Text: cfg.Advanced.Description, VScroll: true},
		LineEdit{AssignTo: rightsLE, Text: cfg.Advanced.Rights, Visible: false},
	}
	return fixedPage("書籍資料", children, func() {
		setWidgetBounds(translatorL, 18, 18, 70, 24)
		setWidgetBounds(*translatorLE, 95, 14, 190, 28)
		setWidgetBounds(isbnL, 310, 18, 55, 24)
		setWidgetBounds(*isbnLE, 365, 14, 190, 28)
		setWidgetBounds(dateL, 575, 18, 75, 24)
		setWidgetBounds(*dateDE, 655, 14, 145, 28)
		setWidgetBounds(publisherL, 18, 58, 70, 24)
		setWidgetBounds(*publisherLE, 95, 54, 190, 28)
		setWidgetBounds(subjectL, 310, 58, 55, 24)
		setWidgetBounds(*subjectCB, 365, 54, 190, 28)
		setWidgetBounds(languageL, 575, 58, 75, 24)
		setWidgetBounds(*languageCB, 655, 54, 145, 28)
		setWidgetBounds(descL, 18, 132, 70, 24)
		setWidgetBounds(*descriptionTE, 95, 96, 705, 132)
		setWidgetBounds(*rightsLE, -1000, -1000, 10, 10)
	})
}

func defaultForceTextCoverChecked(coverPath string, cfg *config.Config) bool {
	if strings.TrimSpace(coverPath) == "" {
		return true
	}
	if cfg == nil {
		return true
	}
	return cfg.Recent.ForceTextCoverBool()
}

func loadTagList() []string {
	for _, path := range settingReadPaths("tagList.txt") {
		tags := readListFile(path)
		if len(tags) > 0 {
			return tags
		}
	}
	if _, err := ensureSettingFile("tagList.txt"); err != nil {
		debugLog("ensure tag list failed: %v", err)
	}
	return nil
}

func saveTagListEntries(entries []string) error {
	cleaned := normalizeTagEntries(entries)
	if len(cleaned) == 0 {
		return nil
	}
	path, err := ensureSettingFile("tagList.txt")
	if err != nil {
		return err
	}
	tags := prependUnique(normalizeTagEntries(readListFile(path)), cleaned...)
	return os.WriteFile(path, []byte(strings.Join(tags, "\n")+"\n"), 0644)
}

func refreshSubjectDropdown(subjectCB *walk.ComboBox, tags []string) {
	if subjectCB == nil {
		return
	}
	text := strings.Join(normalizeTagEntries([]string{subjectCB.Text()}), ", ")
	model := prependUnique(loadTagList(), tags...)
	if err := subjectCB.SetModel(model); err != nil {
		debugLog("refresh subject dropdown failed: %v", err)
	}
	_ = subjectCB.SetText(text)
	_ = subjectCB.SetCurrentIndex(-1)
}

func installSubjectAppendBehavior(subjectCB *walk.ComboBox) {
	if subjectCB == nil {
		return
	}
	currentText := strings.Join(normalizeTagEntries([]string{subjectCB.Text()}), ", ")
	previousText := currentText
	changing := false
	_ = subjectCB.SetText(currentText)
	subjectCB.TextChanged().Attach(func() {
		if changing {
			return
		}
		text := subjectCB.Text()
		if text == currentText {
			return
		}
		previousText = currentText
		currentText = text
	})
	subjectCB.CurrentIndexChanged().Attach(func() {
		if changing {
			return
		}
		selected := comboBoxSelectedText(subjectCB)
		if selected == "" {
			return
		}
		base := currentText
		if strings.EqualFold(strings.TrimSpace(base), strings.TrimSpace(selected)) && strings.TrimSpace(previousText) != "" {
			base = previousText
		}
		merged := appendTagText(base, selected)
		changing = true
		_ = subjectCB.SetText(merged)
		// Walk 會在選取確認後再發送一次通知；此時不可提早清除索引，
		// 否則確認事件可能覆寫剛合併的可編輯文字。
		changing = false
		previousText = base
		currentText = merged
	})
}

func comboBoxSelectedText(cb *walk.ComboBox) string {
	if cb == nil {
		return ""
	}
	idx := cb.CurrentIndex()
	if idx < 0 {
		return ""
	}
	if model, ok := cb.Model().([]string); ok && idx < len(model) {
		return strings.TrimSpace(model[idx])
	}
	return strings.TrimSpace(cb.Text())
}

func appendTagText(base, selected string) string {
	return strings.Join(normalizeTagEntries([]string{base, selected}), ", ")
}

func guiSettingsSubjectTags(settings guiSettings) []string {
	if len(settings.SubjectTags) > 0 {
		return normalizeTagEntries(settings.SubjectTags)
	}
	return splitList(settings.Subject)
}

func guiSettingsSubjectText(settings guiSettings) string {
	return strings.Join(guiSettingsSubjectTags(settings), ", ")
}

func normalizeTagEntries(entries []string) []string {
	var out []string
	for _, entry := range entries {
		out = append(out, splitList(entry)...)
	}
	return prependUnique(nil, out...)
}

func guiSubjectBuildArgs(value string) ([]string, []string) {
	tags := normalizeTagEntries([]string{value})
	args := make([]string, 0, len(tags)*2)
	for _, tag := range tags {
		args = append(args, "-subject", tag)
	}
	return tags, args
}

func prependUnique(items []string, values ...string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, value)
	}
	for _, value := range values {
		add(value)
	}
	for _, item := range items {
		add(item)
	}
	return out
}

func readListFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, splitList(line)...)
	}
	return prependUnique(nil, out...)
}

func comboIndex(items []string, value string) int {
	value = strings.TrimSpace(value)
	for i, item := range items {
		if strings.EqualFold(item, value) {
			return i
		}
	}
	return -1
}

func parseBookDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value != "" {
		if t, err := time.Parse("2006-01-02", value); err == nil {
			return t
		}
	}
	return time.Now()
}

func dateEditText(dateDE *walk.DateEdit) string {
	if dateDE == nil {
		return time.Now().Format("2006-01-02")
	}
	date := dateDE.Date()
	if date.IsZero() {
		return ""
	}
	return date.Format("2006-01-02")
}

func cssTab(cssPathLE **walk.LineEdit, cssTE **walk.TextEdit, cssSkipRB, cssAppendRB, cssOverwriteRB **walk.RadioButton, verticalCB, saveCSSCB **walk.CheckBox, cfg *config.Config, cssMode *int, prefs guiSettings, prefsOK bool, showError func(error), owner walk.Form) TabPage {
	cssText, err := loadCachedCSS(cfg.Recent.SaveCSSBool())
	if err != nil {
		debugLog("load cached css failed: %v", err)
	}
	if prefsOK {
		if text, err := loadCachedCSS(true); err != nil {
			debugLog("load remembered css failed: %v", err)
		} else {
			cssText = text
		}
	}
	if cssMode == nil {
		mode := cfg.Recent.CSSOverwriteInt()
		cssMode = &mode
	}
	saveCSSChecked := cfg.Recent.SaveCSSBool()
	verticalChecked := false
	if prefsOK {
		saveCSSChecked = prefs.SaveCSS
		verticalChecked = prefs.VerticalCSS
	}
	var loadBtn *walk.PushButton
	children := []Widget{
		TextEdit{AssignTo: cssTE, Text: cssText, VScroll: true, HScroll: true},
		RadioButton{AssignTo: cssSkipRB, Text: "略過", Value: 0, OnClicked: func() { *cssMode = 0 }},
		RadioButton{AssignTo: cssAppendRB, Text: "追加", Value: 1, OnClicked: func() { *cssMode = 1 }},
		RadioButton{AssignTo: cssOverwriteRB, Text: "覆蓋", Value: 2, OnClicked: func() { *cssMode = 2 }},
		CheckBox{AssignTo: verticalCB, Text: "追加直排 CSS", Checked: verticalChecked},
		CheckBox{AssignTo: saveCSSCB, Text: "自動保存定制css", Checked: saveCSSChecked},
		PushButton{AssignTo: &loadBtn, Text: "加載css文件", OnClicked: func() {
			defer debugRecover("loadCSS")
			debugLog("action=load_css")
			path, ok, err := openFileDialog(owner, "加載css文件", "CSS files (*.css)|*.css|All files (*.*)|*.*")
			if err != nil {
				showError(err)
				return
			}
			if !ok {
				return
			}
			debugLog("css selected: %q", path)
			data, err := os.ReadFile(path)
			if err != nil {
				showError(err)
				return
			}
			_ = (*cssPathLE).SetText(path)
			_ = (*cssTE).SetText(string(data))
		}},
		LineEdit{AssignTo: cssPathLE, Visible: false},
	}
	return fixedPage("定制 CSS", children, func() {
		setWidgetBounds(*cssTE, 16, 16, 790, 160)
		setWidgetBounds(*cssSkipRB, 16, 198, 65, 26)
		setWidgetBounds(*cssAppendRB, 100, 198, 65, 26)
		setWidgetBounds(*cssOverwriteRB, 185, 198, 65, 26)
		setWidgetBounds(*verticalCB, 270, 198, 110, 26)
		setWidgetBounds(*saveCSSCB, 400, 198, 140, 26)
		setWidgetBounds(loadBtn, 680, 196, 120, 30)
		setWidgetBounds(*cssPathLE, -1000, -1000, 10, 10)
	})
}

func setCSSModeRadio(cssMode int, cssSkipRB, cssAppendRB, cssOverwriteRB *walk.RadioButton) {
	if cssSkipRB != nil {
		cssSkipRB.SetChecked(cssMode == 0)
	}
	if cssAppendRB != nil {
		cssAppendRB.SetChecked(cssMode == 1)
	}
	if cssOverwriteRB != nil {
		cssOverwriteRB.SetChecked(cssMode == 2)
	}
}

func setImagePreview(preview *walk.ImageView, path string) {
	if preview == nil {
		return
	}
	if strings.TrimSpace(path) == "" || isDirPath(path) {
		_ = preview.SetImage(nil)
		return
	}
	img, err := walk.NewImageFromFile(path)
	if err != nil {
		debugLog("image preview failed path=%q err=%v", path, err)
		_ = preview.SetImage(nil)
		return
	}
	_ = preview.SetImage(img)
}

func isDirPath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func imagesTab(imageList **walk.ListBox, imagePreview **walk.ImageView, imageRefLE **walk.LineEdit, imageInputs *[]string, updateImages func(), refreshImageSelection func(), selectedImage func() string, showError func(error), owner walk.Form) TabPage {
	var hintL walk.Window
	var copyBtn *walk.PushButton
	children := []Widget{
		fixedLabel{AssignTo: &hintL, Text: "拖放圖片/目錄至此。雙擊查看，Del刪除。文件不能重名"},
		ListBox{AssignTo: imageList,
			OnCurrentIndexChanged: refreshImageSelection,
			OnItemActivated: func() {
				defer debugRecover("activateImage")
				path := selectedImage()
				if path != "" && !isDirPath(path) {
					debugLog("action=open_image path=%q", path)
					if err := openImageFile(path); err != nil {
						debugLog("open image failed path=%q err=%v", path, err)
						showError(err)
					}
				}
			},
			OnKeyDown: func(key walk.Key) {
				defer debugRecover("imageListKeyDown")
				if key != walk.KeyDelete || *imageList == nil {
					return
				}
				idx := (*imageList).CurrentIndex()
				if idx >= 0 && idx < len(*imageInputs) {
					debugLog("action=delete_image index=%d path=%q", idx, (*imageInputs)[idx])
					*imageInputs = append((*imageInputs)[:idx], (*imageInputs)[idx+1:]...)
					updateImages()
				}
			}},
		ImageView{AssignTo: imagePreview, Mode: ImageViewModeZoom},
		PushButton{AssignTo: &copyBtn, Text: "複製到剪貼板", OnClicked: func() {
			defer debugRecover("copyImageReference")
			path := selectedImage()
			if path == "" || isDirPath(path) {
				return
			}
			debugLog("action=copy_image_reference path=%q", path)
			ref := imageReference(filepath.Base(path))
			_ = (*imageRefLE).SetText(ref)
			if err := copyToClipboard(ref); err != nil {
				showError(err)
			}
		}},
		LineEdit{AssignTo: imageRefLE, ReadOnly: true},
	}
	return fixedPage("插圖", children, func() {
		setWidgetBounds(hintL, 36, 28, 520, 24)
		setWidgetBounds(*imageList, 36, 60, 520, 120)
		setWidgetBounds(*imagePreview, 585, 60, 120, 100)
		setWidgetBounds(copyBtn, 585, 168, 120, 30)
		setWidgetBounds(*imageRefLE, 36, 204, 720, 28)
	})
}

func advancedTab(rawTagLE **walk.LineEdit, flowSizeNE **walk.NumberEdit, emptyStyleCB **walk.ComboBox, enableRawHTMLCB, tocSpaceCB **walk.CheckBox, screenWidthNE, screenHeightNE **walk.NumberEdit, cfg *config.Config) TabPage {
	var silentCB, outputSourceCB *walk.CheckBox
	var modeTitleL, htmlTitleL, outputTitleL, coverTitleL walk.Window
	var emptyL, rawTagL, flowL, kbL, screenL, xL, pxL walk.Window
	children := []Widget{
		fixedLabel{AssignTo: &modeTitleL, Text: "處理模式", TextColor: walk.RGB(80, 80, 80)},
		CheckBox{AssignTo: &silentCB, Text: "靜默模式"},
		fixedLabel{AssignTo: &emptyL, Text: "空章節"},
		ComboBox{AssignTo: emptyStyleCB, Model: []string{"略過", "創建", "子目錄"}, CurrentIndex: cfg.Advanced.EmptyChapterStyleInt()},
		CheckBox{AssignTo: tocSpaceCB, Text: "層級目錄縮進", Checked: cfg.Advanced.TOCSpaceBool()},
		fixedLabel{AssignTo: &htmlTitleL, Text: "HTML 與章節", TextColor: walk.RGB(80, 80, 80)},
		CheckBox{AssignTo: enableRawHTMLCB, Text: "HTML源碼標記", Checked: cfg.Advanced.EnableHTMLRawTagBool()},
		fixedLabel{AssignTo: &rawTagL, Text: "標記"},
		LineEdit{AssignTo: rawTagLE, Text: cfg.Advanced.RawHTMLTag()},
		fixedLabel{AssignTo: &flowL, Text: "HTML文件大小上限"},
		NumberEdit{AssignTo: flowSizeNE, Decimals: 0, Increment: 1, SpinButtonsVisible: true},
		fixedLabel{AssignTo: &kbL, Text: "KB"},
		fixedLabel{AssignTo: &outputTitleL, Text: "輸出", TextColor: walk.RGB(80, 80, 80)},
		CheckBox{AssignTo: &outputSourceCB, Text: "輸出到源目錄"},
		fixedLabel{AssignTo: &coverTitleL, Text: "封面尺寸", TextColor: walk.RGB(80, 80, 80)},
		fixedLabel{AssignTo: &screenL, Text: "屏幕尺寸（寬x高）"},
		NumberEdit{AssignTo: screenWidthNE, Decimals: 0, Increment: 1, SpinButtonsVisible: true},
		fixedLabel{AssignTo: &xL, Text: "X"},
		NumberEdit{AssignTo: screenHeightNE, Decimals: 0, Increment: 1, SpinButtonsVisible: true},
		fixedLabel{AssignTo: &pxL, Text: "px"},
	}
	return fixedPage("高級", children, func() {
		setWidgetBounds(modeTitleL, 36, 18, 90, 24)
		setWidgetBounds(silentCB, 50, 52, 120, 26)
		setWidgetBounds(emptyL, 50, 92, 58, 24)
		setWidgetBounds(*emptyStyleCB, 126, 88, 120, 28)
		setWidgetBounds(*tocSpaceCB, 50, 132, 140, 26)
		setWidgetBounds(outputTitleL, 36, 178, 90, 24)
		setWidgetBounds(outputSourceCB, 50, 212, 140, 26)

		setWidgetBounds(htmlTitleL, 342, 18, 110, 24)
		setWidgetBounds(*enableRawHTMLCB, 356, 52, 135, 26)
		setWidgetBounds(rawTagL, 356, 92, 42, 24)
		setWidgetBounds(*rawTagLE, 410, 88, 70, 28)
		setWidgetBounds(flowL, 356, 132, 150, 24)
		setWidgetBounds(*flowSizeNE, 520, 128, 70, 28)
		setWidgetBounds(kbL, 600, 132, 32, 24)
		setWidgetBounds(coverTitleL, 342, 178, 90, 24)
		setWidgetBounds(screenL, 356, 212, 150, 24)
		setWidgetBounds(*screenWidthNE, 520, 208, 64, 28)
		setWidgetBounds(xL, 594, 212, 22, 24)
		setWidgetBounds(*screenHeightNE, 624, 208, 64, 28)
		setWidgetBounds(pxL, 700, 212, 28, 24)
	})
}

func openFileDialog(owner walk.Form, title, filter string) (string, bool, error) {
	dlg := new(walk.FileDialog)
	dlg.Title = title
	dlg.Filter = filter
	ok, err := dlg.ShowOpen(owner)
	return dlg.FilePath, ok, err
}

func openMultipleDialog(owner walk.Form, title, filter string) ([]string, bool, error) {
	dlg := new(walk.FileDialog)
	dlg.Title = title
	dlg.Filter = filter
	ok, err := dlg.ShowOpenMultiple(owner)
	return dlg.FilePaths, ok, err
}

func saveFileDialog(owner walk.Form, title, filter string) (string, bool, error) {
	dlg := new(walk.FileDialog)
	dlg.Title = title
	dlg.Filter = filter
	ok, err := dlg.ShowSave(owner)
	return dlg.FilePath, ok, err
}

func browseFolderDialog(owner walk.Form, title string) (string, bool, error) {
	dlg := new(walk.FileDialog)
	dlg.Title = title
	ok, err := dlg.ShowBrowseFolder(owner)
	return dlg.FilePath, ok, err
}

func openTextFileWithEditor(path, editor string) error {
	if strings.TrimSpace(editor) == "" {
		return openImageFile(path)
	}
	cmd := exec.Command(editor, path)
	return cmd.Start()
}

func defaultGUIOutputPath(inputPath string) string {
	name := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)) + ".epub"
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join(filepath.Dir(inputPath), name)
	}
	return filepath.Join(filepath.Dir(exe), name)
}

func detectGUIChapters(inputPath, configPath, title string, mode int, regexValue, splitCountValue, chapterPlanPath string) ([]book.ChapterMark, error) {
	cfg, _, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(title) == "" {
		title = strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	}
	text, _, err := textio.DecodeFile(inputPath, "auto")
	if err != nil {
		return nil, err
	}
	chapterRegex := ""
	splitCount := 0
	switch mode {
	case 2:
		splitCount, _ = strconv.Atoi(strings.TrimSpace(splitCountValue))
	case 3:
		if strings.TrimSpace(chapterPlanPath) != "" {
			return book.LoadChapterPlan(chapterPlanPath)
		}
	default:
		chapterRegex = strings.TrimSpace(regexValue)
		if chapterRegex == "" {
			chapterRegex = cfg.ChapterRegex()
		}
	}
	return book.DetectChapterPlan(text, book.TextOptions{
		Title:        title,
		ChapterRegex: chapterRegex,
		SplitCount:   splitCount,
		TOCSpace:     cfg.Advanced.TOCSpaceBool(),
	})
}

func simpleChapterRegexFromUI(prefix, numberKind, unit, extra string, lineStartSpace bool) string {
	cfg := config.Default()
	cfg.Recent.SimpleRegP1 = strings.TrimSpace(prefix)
	cfg.Recent.SimpleRegP2 = simpleNumberModeFromUI(numberKind)
	cfg.Recent.SimpleRegP3 = strings.TrimSpace(unit)
	cfg.Recent.SimpleRegExt = strings.TrimSpace(extra)
	if lineStartSpace {
		cfg.Recent.SimpleLeadingSpace = "1"
	} else {
		cfg.Recent.SimpleLeadingSpace = "0"
	}
	return cfg.SimpleChapterRegex()
}

func simpleNumberModeFromUI(value string) string {
	switch strings.TrimSpace(value) {
	case "純阿拉伯數字":
		return "1"
	case "純中文數字":
		return "2"
	default:
		return "0"
	}
}

func guiTextLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}

func writeTempCSS(css string) (string, func(), error) {
	file, err := os.CreateTemp("", "ezpub-*.css")
	if err != nil {
		return "", nil, err
	}
	path := file.Name()
	if _, err := file.WriteString(css); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", nil, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func appendUniquePaths(paths []string, additions ...string) []string {
	seen := map[string]bool{}
	for _, path := range paths {
		seen[strings.ToLower(filepath.Clean(path))] = true
	}
	for _, path := range additions {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" {
			continue
		}
		key := strings.ToLower(path)
		if !seen[key] {
			paths = append(paths, path)
			seen[key] = true
		}
	}
	return paths
}

func splitList(value string) []string {
	return splitCommaList(value)
}

func ensureExt(path, ext string) string {
	if path == "" || strings.EqualFold(filepath.Ext(path), ext) {
		return path
	}
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}

func intValue(value string, fallback int) int {
	i, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return i
}

func floatValue(value string, fallback float64) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return fallback
	}
	return f
}

func intText(value int) string {
	if value <= 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
