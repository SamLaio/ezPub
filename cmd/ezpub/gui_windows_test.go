//go:build windows

package main

import (
	"reflect"
	"testing"

	"ezpub/internal/config"
)

func TestInitialGUIFontType(t *testing.T) {
	for _, tt := range []struct {
		name   string
		stored int
		loaded bool
		want   int
	}{
		{"首次啟動", 0, false, 3},
		{"未載入的資料不採用", 2, false, 3},
		{"保留自訂字型", 1, true, 1},
		{"保留內嵌字型", 2, true, 2},
		{"保留閱讀器字型", 3, true, 3},
		{"缺少選項", 0, true, 3},
		{"無效選項", 4, true, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := initialGUIFontType(guiSettings{FontType: tt.stored}, tt.loaded); got != tt.want {
				t.Fatalf("font type = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalizeTagEntriesSplitsCommaAndTrims(t *testing.T) {
	got := normalizeTagEntries([]string{" t1, t2 ", "t1， t3; ; t4"})
	want := []string{"t1", "t2", "t3", "t4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}

func TestAppendTagText(t *testing.T) {
	tests := []struct {
		base     string
		selected string
		want     string
	}{
		{selected: " t2 ", want: "t2"},
		{base: "t1", selected: "t2", want: "t1, t2"},
		{base: "t1, t2", selected: "t2", want: "t1, t2"},
	}
	for _, tt := range tests {
		if got := appendTagText(tt.base, tt.selected); got != tt.want {
			t.Fatalf("appendTagText(%q, %q) = %q, want %q", tt.base, tt.selected, got, tt.want)
		}
	}
}

func TestGUISubjectBuildArgsSplitsManualInput(t *testing.T) {
	tags, args := guiSubjectBuildArgs(" 小說，科幻; 反派, 小說 ")
	if want := []string{"小說", "科幻", "反派"}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("tags = %#v, want %#v", tags, want)
	}
	if want := []string{"-subject", "小說", "-subject", "科幻", "-subject", "反派"}; !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestParseGUIDebugArgsDefaultsOff(t *testing.T) {
	enabled, args := parseGUIDebugArgs([]string{"book.txt"})
	if enabled {
		t.Fatal("debug should be disabled by default")
	}
	if !reflect.DeepEqual(args, []string{"book.txt"}) {
		t.Fatalf("args = %#v, want book.txt", args)
	}

	enabled, args = parseGUIDebugArgs([]string{"-debug", "book.txt"})
	if !enabled {
		t.Fatal("debug should be enabled by -debug")
	}
	if !reflect.DeepEqual(args, []string{"book.txt"}) {
		t.Fatalf("args = %#v, want book.txt", args)
	}
}

func TestNormalizeCommaListText(t *testing.T) {
	if got, want := normalizeCommaListText(" a, b， a ; c "), "a, b, c"; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}

func TestGUISettingsSubjectTextUsesTags(t *testing.T) {
	settings := guiSettings{Subject: "legacy1, legacy2", SubjectTags: []string{" t1, t2 ", "t3"}}
	if got, want := guiSettingsSubjectText(settings), "t1, t2, t3"; got != want {
		t.Fatalf("subject text = %q, want %q", got, want)
	}
}

func TestDefaultForceTextCoverCheckedWhenCoverEmpty(t *testing.T) {
	cfg := config.Default()
	cfg.Recent.ForceTextCover = "0"
	if !defaultForceTextCoverChecked("", cfg) {
		t.Fatal("empty cover path should default to text cover")
	}
	if defaultForceTextCoverChecked("cover.jpg", cfg) {
		t.Fatal("non-empty cover path should keep stored force text cover setting")
	}
}
