//go:build windows

package main

import (
	"reflect"
	"testing"
)

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
