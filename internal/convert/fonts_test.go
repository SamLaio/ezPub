package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("EZPUB_TEST_SUBSETTER") == "1" {
		for _, arg := range os.Args[1:] {
			if strings.HasPrefix(arg, "--output-file=") {
				if err := os.WriteFile(strings.TrimPrefix(arg, "--output-file="), []byte("subset"), 0600); err != nil {
					os.Exit(1)
				}
				os.Exit(0)
			}
		}
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestSubsetFontsPreservesCSSFilenames(t *testing.T) {
	t.Setenv("EZPUB_TEST_SUBSETTER", "1")
	tool, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var fonts []string
	for _, dir := range []string{t.TempDir(), t.TempDir()} {
		path := filepath.Join(dir, "body.otf")
		if err := os.WriteFile(path, []byte("original-font-fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		fonts = append(fonts, path)
	}
	got, cleanup, err := SubsetFonts(fonts, "正文", tool)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if len(got) != 2 || got[0] == got[1] {
		t.Fatalf("字型暫存路徑衝突：%v", got)
	}
	for _, path := range got {
		data, err := os.ReadFile(path)
		if filepath.Base(path) != "body.otf" || err != nil || string(data) != "subset" {
			t.Fatalf("子集化改變CSS檔名或未產生子集：%s %v", path, err)
		}
	}
}

func TestSubsetCharacters(t *testing.T) {
	got := subsetCharacters("乙 甲\n甲\t乙")
	want := " 乙甲"
	if got != want {
		t.Fatalf("subset characters = %q, want %q", got, want)
	}
}
