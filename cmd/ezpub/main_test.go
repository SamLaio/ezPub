package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ezpub/internal/config"
)

func TestDefaultOutputPathUsesEasyPubOutputFolder(t *testing.T) {
	cfg := config.Default()
	cfg.Recent.OutputFolder = filepath.Join("D:", "ebooks")
	cfg.Advanced.OutputToSrc = "0"

	got := defaultOutputPath(filepath.Join("C:", "books", "novel.txt"), ".epub", cfg)
	want := filepath.Join("D:", "ebooks", "novel.epub")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
}

func TestDefaultOutputPathCanUseSourceDirectory(t *testing.T) {
	cfg := config.Default()
	cfg.Recent.OutputFolder = filepath.Join("D:", "ebooks")
	cfg.Advanced.OutputToSrc = "1"

	got := defaultOutputPath(filepath.Join("C:", "books", "novel.txt"), ".epub", cfg)
	want := filepath.Join("C:", "books", "novel.epub")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
}

func TestSplitTitleAuthor(t *testing.T) {
	title, author := splitTitleAuthor("巴黎茶花女遺事 - 小仲馬")
	if title != "巴黎茶花女遺事" || author != "小仲馬" {
		t.Fatalf("title author = %q %q", title, author)
	}
}

func TestBuildRequiresEPUBOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "novel.txt")
	if err := os.WriteFile(input, []byte("第一章\n內容"), 0644); err != nil {
		t.Fatal(err)
	}
	err := runBuild([]string{"-o", filepath.Join(dir, "novel.txt"), input})
	if err == nil || !strings.Contains(err.Error(), "output path must end with .epub") {
		t.Fatalf("runBuild error = %v, want .epub requirement", err)
	}
}

func TestBuildReaderDefaultFontIgnoresFontCSS(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "novel.txt")
	output := filepath.Join(dir, "novel.epub")
	font := filepath.Join(dir, "reader.ttf")
	if err := os.WriteFile(input, []byte("第一章\n內容"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(font, []byte("not a real font but should be ignored"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runBuild([]string{"-o", output, "-font-type", "3", "-font", font, "-font-source", "res:///system/fonts/DroidSansFallback.ttf", input}); err != nil {
		t.Fatal(err)
	}
	css := readEPUBEntry(t, output, "EPUB/styles/style.css")
	for _, bad := range []string{"@font-face", "ezpub-reader-font", "font-family:"} {
		if strings.Contains(css, bad) {
			t.Fatalf("reader default font should not emit %q:\n%s", bad, css)
		}
	}
	if got := readEPUBEntry(t, output, "EPUB/content.opf"); strings.Contains(got, "reader.ttf") {
		t.Fatalf("reader default font should not embed font file:\n%s", got)
	}
}

func TestBuildCustomReaderFontSourceEmitsFontCSS(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "novel.txt")
	output := filepath.Join(dir, "novel.epub")
	source := "res:///system/fonts/DroidSansFallback.ttf"
	if err := os.WriteFile(input, []byte("第一章\n內容"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runBuild([]string{"-o", output, "-font-type", "1", "-font-source", source, input}); err != nil {
		t.Fatal(err)
	}
	css := readEPUBEntry(t, output, "EPUB/styles/style.css")
	for _, want := range []string{
		`@font-face { font-family: "ezpub-reader-font";`,
		`url("res:///system/fonts/DroidSansFallback.ttf")`,
		`font-family: "ezpub-reader-font", serif;`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("custom reader font should emit %q:\n%s", want, css)
		}
	}
	if got := readEPUBEntry(t, output, "EPUB/content.opf"); strings.Contains(got, "DroidSansFallback.ttf") {
		t.Fatalf("custom reader font source should not embed font file:\n%s", got)
	}
}

func TestApplyBuildOverridesFontType(t *testing.T) {
	cfg := config.Default()
	applyBuildOverrides(cfg, []string{"-font-type", "3"}, buildOverrides{fontType: 3})
	if got := cfg.Recent.FontTypeInt(); got != 3 {
		t.Fatalf("font type = %d, want 3", got)
	}
}

func TestApplyBuildOverridesPageMarginsUseSharedUnit(t *testing.T) {
	cfg := config.Default()
	applyBuildOverrides(cfg,
		[]string{"-margin-top", "1", "-margin-bottom", "2", "-margin-left", "3", "-margin-right", "4", "-margin-unit", "2"},
		buildOverrides{marginTop: 1, marginBottom: 2, marginLeft: 3, marginRight: 4, marginUnit: 2},
	)
	layout := cfg.LayoutOptions()
	if layout.MarginTop.Value != 1 || layout.MarginBottom.Value != 2 || layout.MarginLeft.Value != 3 || layout.MarginRight.Value != 4 {
		t.Fatalf("margins = %#v", layout)
	}
	for name, got := range map[string]string{
		"top":    layout.MarginTop.Unit,
		"bottom": layout.MarginBottom.Unit,
		"left":   layout.MarginLeft.Unit,
		"right":  layout.MarginRight.Unit,
	} {
		if got != "em" {
			t.Fatalf("%s margin unit = %q, want em", name, got)
		}
	}
}

func TestCollectImageAssetsRejectsDuplicateBasenames(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.MkdirAll(a, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(b, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a, "pic.jpg"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b, "pic.jpg"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := collectImageAssets([]string{dir}); err == nil {
		t.Fatal("collectImageAssets should reject duplicate basenames")
	}
}

func TestImageReference(t *testing.T) {
	got := imageReference("images.jpg")
	want := `##<div class="centeredimage"><img src="images/images.jpg" alt="images.jpg" class="attpic" /></div>`
	if got != want {
		t.Fatalf("image reference = %q, want %q", got, want)
	}
}

func TestCollectFixtureImages(t *testing.T) {
	assets, err := collectImageAssets([]string{filepath.Join("..", "..", "test_file", "images.jpg")})
	if err != nil {
		t.Skipf("fixture image not available: %v", err)
	}
	if len(assets) != 1 || assets[0].Name != "images.jpg" {
		t.Fatalf("assets = %#v, want images.jpg", assets)
	}
}

func readEPUBEntry(t *testing.T, path, name string) string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	t.Fatalf("epub entry %s not found", name)
	return ""
}
