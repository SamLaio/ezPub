package config

import (
	"regexp"
	"testing"
)

func TestChapterRegexModes(t *testing.T) {
	cfg := Default()
	cfg.Recent.SplitMode = "0"
	cfg.Recent.SimpleRegP1 = "第"
	cfg.Recent.SimpleRegP2 = "0"
	cfg.Recent.SimpleRegP3 = "章"
	cfg.Recent.SimpleRegExt = `^\s*(序言|後記)`
	re, err := regexp.Compile(cfg.ChapterRegex())
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("第一章 開始") {
		t.Fatal("simple regex did not match Chinese chapter heading")
	}
	if !re.MatchString("後記") {
		t.Fatal("simple regex did not include extra expression")
	}

	cfg.Recent.SplitMode = "2"
	if got := cfg.ChapterRegex(); got != "" {
		t.Fatalf("chapter regex = %q, want empty for length split", got)
	}
}

func TestEasyPubSplitCountOnlyForLengthMode(t *testing.T) {
	cfg := Default()
	cfg.Recent.SplitMode = "2"
	cfg.Recent.SplitCount = "3"
	if got := cfg.SplitCount(); got != 3 {
		t.Fatalf("split count = %d, want 3", got)
	}

	cfg.Recent.SplitMode = "1"
	if got := cfg.SplitCount(); got != 0 {
		t.Fatalf("split count = %d, want 0", got)
	}
}

func TestEmbeddedFonts(t *testing.T) {
	recent := RecentOptions{FontType: "2", FontEmbedded: "a.ttf\r\nb.otf;c.ttf"}
	got := recent.EmbeddedFonts()
	if len(got) != 3 {
		t.Fatalf("font count = %d, want 3: %#v", len(got), got)
	}
}

func TestEmbeddedFontsRequireEmbeddedFontType(t *testing.T) {
	recent := RecentOptions{FontType: "1", FontEmbedded: "a.ttf"}
	if got := recent.EmbeddedFonts(); len(got) != 0 {
		t.Fatalf("font count = %d, want 0: %#v", len(got), got)
	}
}

func TestEasyPubFontTypes(t *testing.T) {
	tests := []struct {
		fontType   string
		machine    bool
		customized bool
		embedded   bool
		reader     bool
	}{
		{fontType: "0", machine: true},
		{fontType: "1", customized: true},
		{fontType: "2", embedded: true},
		{fontType: "3", reader: true},
	}
	for _, tt := range tests {
		recent := RecentOptions{FontType: tt.fontType}
		if recent.UseMachineFontsBool() != tt.machine {
			t.Fatalf("fonttype %s machine = %v, want %v", tt.fontType, recent.UseMachineFontsBool(), tt.machine)
		}
		if recent.UseCustomizedFontsBool() != tt.customized {
			t.Fatalf("fonttype %s customized = %v, want %v", tt.fontType, recent.UseCustomizedFontsBool(), tt.customized)
		}
		if recent.UseEmbeddedFontsBool() != tt.embedded {
			t.Fatalf("fonttype %s embedded = %v, want %v", tt.fontType, recent.UseEmbeddedFontsBool(), tt.embedded)
		}
		if recent.UseDefaultReaderFontsBool() != tt.reader {
			t.Fatalf("fonttype %s reader default = %v, want %v", tt.fontType, recent.UseDefaultReaderFontsBool(), tt.reader)
		}
	}
}

func TestEasyPubInvalidFontTypeFallsBackToReaderDefault(t *testing.T) {
	recent := RecentOptions{FontType: "99"}
	if got := recent.FontTypeInt(); got != 3 {
		t.Fatalf("font type = %d, want 3", got)
	}
	if !recent.UseDefaultReaderFontsBool() {
		t.Fatal("invalid font type should use reader default")
	}
}

func TestEasyPubBooleanOptions(t *testing.T) {
	recent := RecentOptions{ForceTextCover: "0", FontSubsetting: "1", AddSpace: "1", AddSpaceCount: "3"}
	if recent.ForceTextCoverBool() {
		t.Fatal("force text cover = true, want false")
	}
	if !recent.FontSubsettingBool() {
		t.Fatal("font subsetting = false, want true")
	}
	if !recent.AddSpaceBool() {
		t.Fatal("add space = false, want true")
	}
	if got := recent.AddSpaceCountInt(); got != 3 {
		t.Fatalf("add space count = %d, want 3", got)
	}

	advanced := AdvancedOptions{OutputToSrc: "1", ForceEmpty: "0"}
	if !advanced.OutputToSrcBool() {
		t.Fatal("output to source = false, want true")
	}
	if advanced.ForceEmptyChapterBool() {
		t.Fatal("force empty chapter = true, want false")
	}
}

func TestEasyPubEmptyChapterStyle(t *testing.T) {
	if !(AdvancedOptions{ForceEmpty: "1", EmptyStyle: "0"}).SkipEmptyChaptersBool() {
		t.Fatal("empty style 0 should skip empty chapters")
	}
	if (AdvancedOptions{ForceEmpty: "1", EmptyStyle: "1"}).SkipEmptyChaptersBool() {
		t.Fatal("empty style 1 should create empty chapters")
	}
	if !(AdvancedOptions{ForceEmpty: "1", EmptyStyle: "2"}).EmptyChaptersAsSubdirectoryBool() {
		t.Fatal("empty style 2 should create TOC-only subdirectory entries")
	}
}

func TestEasyPubLayoutUnits(t *testing.T) {
	cfg := Default()
	cfg.Recent.Top = "10"
	cfg.Recent.PageTopUnit = "0"
	cfg.Recent.MarginTop = "1.5"
	cfg.Recent.MarginTopUnit = "2"
	layout := cfg.LayoutOptions()
	if got := layout.MarginTop.String(); got != "10px" {
		t.Fatalf("top margin = %q, want 10px", got)
	}
	if got := layout.ParagraphSpacing.String(); got != "1.5em" {
		t.Fatalf("paragraph spacing = %q, want 1.5em", got)
	}
}
