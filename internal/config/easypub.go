package config

import (
	"encoding/xml"
	"os"
	"regexp"
	"strconv"
	"strings"

	"ezpub/internal/book"
	"ezpub/internal/lang"
)

type Config struct {
	XMLName        xml.Name        `xml:"EasyPubConfig"`
	XMLVersion     string          `xml:"XMLVersion"`
	EReadersConfig string          `xml:"eReadersConfig"`
	MyRegExp       MyRegExp        `xml:"MyRegExp"`
	Recent         RecentOptions   `xml:"RecentOptions"`
	Advanced       AdvancedOptions `xml:"AdvancedOptions"`
}

type MyRegExp struct {
	Additional RegList `xml:"AdditionalReg"`
	Full       RegList `xml:"FullReg"`
}

type RegList struct {
	Data []string `xml:"data"`
}

type RecentOptions struct {
	SplitMode          string `xml:"splitmode"`
	SplitCount         string `xml:"splitcount"`
	Top                string `xml:"top"`
	Bottom             string `xml:"bottom"`
	Left               string `xml:"left"`
	Right              string `xml:"right"`
	LineHeight         string `xml:"lineheight"`
	FontSize           string `xml:"fontsize"`
	Indent             string `xml:"indent"`
	MarginTop          string `xml:"margintop"`
	RemoveBlankLine    string `xml:"removeblankline"`
	FontType           string `xml:"fonttype"`
	MachineID          string `xml:"machineid"`
	FontCustomized     string `xml:"font_customized"`
	FontEmbedded       string `xml:"font_embedded"`
	FontSubsetting     string `xml:"font_subsetting"`
	Editor             string `xml:"editor"`
	OutputFolder       string `xml:"outputfolder"`
	CSSOverwrite       string `xml:"cssoverwrite"`
	ForceTextCover     string `xml:"forcetextcover"`
	AddSpace           string `xml:"addspace"`
	SaveCSS            string `xml:"savecss"`
	SimpleRegP1        string `xml:"simple_reg_p1"`
	SimpleRegP2        string `xml:"simple_reg_p2"`
	SimpleRegP3        string `xml:"simple_reg_p3"`
	SimpleRegExt       string `xml:"simple_reg_ext"`
	FullReg            string `xml:"full_reg"`
	SimpleLeadingSpace string `xml:"simple_reg_leadingspace"`
	TextAlign          string `xml:"textalign"`
	AddSpaceCount      string `xml:"addspacecount"`
	CoverStyle         string `xml:"coverstyle"`
	TitleFont          string `xml:"titlefont"`
	AuthorFont         string `xml:"authorfont"`
	PageTopUnit        string `xml:"pagetopunit"`
	PageBottomUnit     string `xml:"pagebottomunit"`
	PageLeftUnit       string `xml:"pageleftunit"`
	PageRightUnit      string `xml:"pagerightunit"`
	MarginTopUnit      string `xml:"margintopunit"`
}

type AdvancedOptions struct {
	SilentMode    string `xml:"silentmode"`
	Description   string `xml:"description"`
	Publisher     string `xml:"publisher"`
	Date          string `xml:"date"`
	Identifier    string `xml:"identifier"`
	Rights        string `xml:"rights"`
	EnableHTMLRaw string `xml:"enable_htmlrawtag"`
	HTMLRawTag    string `xml:"htmlrawtag"`
	EnableTempDir string `xml:"enable_tempdir"`
	TempDir       string `xml:"tempdir"`
	FlowSize      string `xml:"flowsize"`
	ScreenWidth   string `xml:"screenwidth"`
	ScreenHeight  string `xml:"screenheight"`
	CoverStyle    string `xml:"coverstyle"`
	TOCSpace      string `xml:"tocspace"`
	ForceEmpty    string `xml:"forceemptychapter"`
	OutputToSrc   string `xml:"outputtosrc"`
	EmptyStyle    string `xml:"emptychapterstyle"`
	AlwaysOnTop   string `xml:"alwaysontop"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := xml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.fillDefaults()
	return &cfg, nil
}

func Default() *Config {
	cfg := &Config{}
	cfg.fillDefaults()
	return cfg
}

func (c *Config) fillDefaults() {
	if c.Recent.FullReg == "" {
		c.Recent.FullReg = lang.DefaultChapterRegex
	}
	if c.Recent.LineHeight == "" {
		c.Recent.LineHeight = "130"
	}
	if c.Recent.FontSize == "" {
		c.Recent.FontSize = "100"
	}
	if c.Recent.Indent == "" {
		c.Recent.Indent = "2"
	}
	if c.Recent.RemoveBlankLine == "" {
		c.Recent.RemoveBlankLine = "1"
	}
	if c.Advanced.HTMLRawTag == "" {
		c.Advanced.HTMLRawTag = "##"
	}
}

func (c *Config) ChapterRegex() string {
	switch intDefault(c.Recent.SplitMode, 1) {
	case 0:
		return c.SimpleChapterRegex()
	case 2:
		return ""
	}
	if strings.TrimSpace(c.Recent.FullReg) != "" {
		return strings.TrimSpace(c.Recent.FullReg)
	}
	if len(c.MyRegExp.Full.Data) > 0 {
		return strings.TrimSpace(c.MyRegExp.Full.Data[0])
	}
	return lang.DefaultChapterRegex
}

func (c *Config) SimpleChapterRegex() string {
	prefix := strings.TrimSpace(c.Recent.SimpleRegP1)
	if prefix == "" {
		prefix = "第"
	}
	suffix := strings.TrimSpace(c.Recent.SimpleRegP3)
	if suffix == "" {
		suffix = "章"
	}
	number := simpleNumberPattern(c.Recent.SimpleRegP2)
	leading := "^"
	if intDefault(c.Recent.SimpleLeadingSpace, 0) != 0 {
		leading += `\s*`
	}

	parts := []string{leading + regexp.QuoteMeta(prefix) + number + regexp.QuoteMeta(suffix) + `.*`}
	if extra := strings.TrimSpace(c.Recent.SimpleRegExt); extra != "" {
		parts = append(parts, extra)
	}
	return strings.Join(parts, "|")
}

func (c *Config) LayoutOptions() book.LayoutOptions {
	return book.LayoutOptions{
		LineHeightPercent: intDefault(c.Recent.LineHeight, 130),
		FontSizePercent:   intDefault(c.Recent.FontSize, 100),
		IndentEm:          floatDefault(c.Recent.Indent, 2),
		TextAlign:         intDefault(c.Recent.TextAlign, 0),
		MarginTop:         cssLength(c.Recent.Top, c.Recent.PageTopUnit),
		MarginBottom:      cssLength(c.Recent.Bottom, c.Recent.PageBottomUnit),
		MarginLeft:        cssLength(c.Recent.Left, c.Recent.PageLeftUnit),
		MarginRight:       cssLength(c.Recent.Right, c.Recent.PageRightUnit),
		ParagraphSpacing:  cssLength(c.Recent.MarginTop, c.Recent.MarginTopUnit),
	}
}

func (c *Config) CoverOptions() book.CoverOptions {
	return book.CoverOptions{
		Style:          intDefault(c.Recent.CoverStyle, intDefault(c.Advanced.CoverStyle, 0)),
		TitleFontPX:    intDefault(c.Recent.TitleFont, 50),
		AuthorFontPX:   intDefault(c.Recent.AuthorFont, 25),
		ScreenHeightPX: intDefault(c.Advanced.ScreenHeight, 720),
	}
}

func simpleNumberPattern(mode string) string {
	switch intDefault(mode, 0) {
	case 1:
		return `[0123456789０１２３４５６７８９]+`
	case 2:
		return `[一二三四五六七八九十零〇百千两兩]+`
	default:
		return `[0123456789０１２３４５６７８９一二三四五六七八九十零〇百千两兩]+`
	}
}

func (c *Config) SplitCount() int {
	if intDefault(c.Recent.SplitMode, 1) != 2 {
		return 0
	}
	return intDefault(c.Recent.SplitCount, 0)
}

func (r RecentOptions) RemoveBlankLineBool() bool {
	return intDefault(r.RemoveBlankLine, 1) != 0
}

func (r RecentOptions) CSSOverwriteInt() int {
	return intDefault(r.CSSOverwrite, 0)
}

func (r RecentOptions) EmbeddedFonts() []string {
	if !r.UseEmbeddedFontsBool() {
		return nil
	}
	return splitFontList(r.FontEmbedded)
}

func (r RecentOptions) UseMachineFontsBool() bool {
	return r.FontTypeInt() == 0
}

func (r RecentOptions) UseCustomizedFontsBool() bool {
	return r.FontTypeInt() == 1
}

func (r RecentOptions) UseEmbeddedFontsBool() bool {
	return r.FontTypeInt() == 2
}

func (r RecentOptions) UseDefaultReaderFontsBool() bool {
	return r.FontTypeInt() == 3
}

func (r RecentOptions) FontTypeInt() int {
	value := intDefault(r.FontType, 3)
	if value < 0 || value > 3 {
		return 3
	}
	return value
}

func (r RecentOptions) FontSubsettingBool() bool {
	return intDefault(r.FontSubsetting, 0) != 0
}

func (r RecentOptions) ForceTextCoverBool() bool {
	return intDefault(r.ForceTextCover, 1) != 0
}

func (r RecentOptions) AddSpaceBool() bool {
	return intDefault(r.AddSpace, 0) != 0
}

func (r RecentOptions) AddSpaceCountInt() int {
	return intDefault(r.AddSpaceCount, 2)
}

func (r RecentOptions) SaveCSSBool() bool {
	return intDefault(r.SaveCSS, 0) != 0
}

func (a AdvancedOptions) EnableHTMLRawTagBool() bool {
	return intDefault(a.EnableHTMLRaw, 0) != 0
}

func (a AdvancedOptions) TOCSpaceBool() bool {
	return intDefault(a.TOCSpace, 1) != 0
}

func (a AdvancedOptions) ForceEmptyChapterBool() bool {
	return intDefault(a.ForceEmpty, 1) != 0
}

func (a AdvancedOptions) EmptyChapterStyleInt() int {
	style := intDefault(a.EmptyStyle, 1)
	if style < 0 || style > 2 {
		return 1
	}
	return style
}

func (a AdvancedOptions) SkipEmptyChaptersBool() bool {
	return !a.ForceEmptyChapterBool() || a.EmptyChapterStyleInt() == 0
}

func (a AdvancedOptions) EmptyChaptersAsSubdirectoryBool() bool {
	return a.ForceEmptyChapterBool() && a.EmptyChapterStyleInt() == 2
}

func (a AdvancedOptions) FlowSizeKB() int {
	return intDefault(a.FlowSize, 0)
}

func (a AdvancedOptions) RawHTMLTag() string {
	if strings.TrimSpace(a.HTMLRawTag) == "" {
		return "##"
	}
	return a.HTMLRawTag
}

func (a AdvancedOptions) OutputToSrcBool() bool {
	return intDefault(a.OutputToSrc, 0) != 0
}

func intDefault(s string, fallback int) int {
	i, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return i
}

func floatDefault(s string, fallback float64) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return fallback
	}
	return f
}

func cssLength(value, unit string) book.CSSLength {
	return book.CSSLength{
		Value: floatDefault(value, 0),
		Unit:  cssUnit(unit),
	}
}

func cssUnit(unit string) string {
	switch intDefault(unit, 0) {
	case 1:
		return "%"
	case 2:
		return "em"
	default:
		return "px"
	}
}
