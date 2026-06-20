//go:build windows

package main

import (
	"testing"

	"ezpub/internal/book"
	"ezpub/internal/config"
)

func TestAddChapterBySourceLineUsesTextLineAndSorts(t *testing.T) {
	model := &chapterEditorModel{rows: []chapterEditorRow{
		{Index: 1, Title: "正文", Line: 1, Level: 1},
		{Index: 2, Title: "第一節", Line: 9, Level: 1},
		{Index: 3, Title: "第二節", Line: 28, Level: 1},
	}}
	model.renumber()

	sourceLines := make([]string, 30)
	sourceLines[9] = "這是原始 TXT 第十行"

	idx, added, err := addChapterBySourceLine(model, sourceLines, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("added = false, want true")
	}
	if idx != 2 {
		t.Fatalf("inserted index = %d, want 2", idx)
	}
	if got := model.rows[2].Title; got != "這是原始 TXT 第十行" {
		t.Fatalf("title = %q", got)
	}
	if got := model.rows[2].Line; got != 10 {
		t.Fatalf("line = %d, want 10", got)
	}
	if got := []int{model.rows[0].Line, model.rows[1].Line, model.rows[2].Line, model.rows[3].Line}; got[0] != 1 || got[1] != 9 || got[2] != 10 || got[3] != 28 {
		t.Fatalf("line order = %v, want [1 9 10 28]", got)
	}
}

func TestChapterEditorModelShowsIndentedLevelInIndexColumn(t *testing.T) {
	model := &chapterEditorModel{rows: []chapterEditorRow{
		{Index: 1, Title: "第一節", Line: 9, Level: 1},
		{Index: 2, Title: "第二節", Line: 28, Level: 2},
		{Index: 3, Title: "第三節", Line: 39, Level: 3},
	}}

	if got := model.Value(0, 0); got != "1" {
		t.Fatalf("level 1 index = %v, want 1", got)
	}
	if got := model.Value(1, 0); got != "+2" {
		t.Fatalf("level 2 index = %v, want +2", got)
	}
	if got := model.Value(2, 0); got != "++3" {
		t.Fatalf("level 3 index = %v, want ++3", got)
	}
	if got := model.Value(1, 1); got != "第二節" {
		t.Fatalf("title = %v, want unindented title", got)
	}
}

func TestAdjustChapterLevelBounds(t *testing.T) {
	model := &chapterEditorModel{rows: []chapterEditorRow{
		{Index: 1, Title: "第一節", Line: 9, Level: 1},
		{Index: 2, Title: "第二節", Line: 28, Level: 5},
	}}

	if adjustChapterLevel(model, 0, -1) {
		t.Fatal("level 1 should not decrease")
	}
	if got := model.rows[0].Level; got != 1 {
		t.Fatalf("level = %d, want 1", got)
	}
	if !adjustChapterLevel(model, 0, 1) {
		t.Fatal("level should increase")
	}
	if got := model.rows[0].Level; got != 2 {
		t.Fatalf("level = %d, want 2", got)
	}
	if !adjustChapterLevel(model, 1, 3) {
		t.Fatal("level should clamp upward")
	}
	if got := model.rows[1].Level; got != 6 {
		t.Fatalf("level = %d, want 6", got)
	}
	if adjustChapterLevel(model, 1, 1) {
		t.Fatal("level 6 should not increase")
	}
}

func TestAddChapterBySourceLineRestoresExistingLine(t *testing.T) {
	model := &chapterEditorModel{rows: []chapterEditorRow{
		{Index: 1, Title: "第一節", Line: 9, Level: 1, Deleted: true},
	}}
	model.renumber()

	idx, added, err := addChapterBySourceLine(model, []string{"一", "二", "三", "四", "五", "六", "七", "八", "第一節"}, 9)
	if err != nil {
		t.Fatal(err)
	}
	if added {
		t.Fatal("added = true, want false for existing line")
	}
	if idx != 0 {
		t.Fatalf("index = %d, want 0", idx)
	}
	if model.rows[0].Deleted {
		t.Fatal("existing row should be restored")
	}
	if len(model.rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(model.rows))
	}
}

func TestSimpleChapterRegexFromUIUsesSelectedUnit(t *testing.T) {
	re := simpleChapterRegexFromUI("第", "混合型數字", "章", "", false)
	marks, err := book.DetectChapterPlan("第一節\n內容\n第一章\n內容\n第二節\n內容", book.TextOptions{
		Title:        "測試",
		ChapterRegex: re,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(marks) != 2 {
		t.Fatalf("marks = %#v, want preface + 第一章 only", marks)
	}
	if marks[1].Title != "第一章" {
		t.Fatalf("matched title = %q, want 第一章", marks[1].Title)
	}
}

func TestSimpleChapterRegexFromUIMatchesConfigLogic(t *testing.T) {
	cfg := config.Default()
	cfg.Recent.SimpleRegP1 = "第"
	cfg.Recent.SimpleRegP2 = "0"
	cfg.Recent.SimpleRegP3 = "節"
	cfg.Recent.SimpleRegExt = `^\s*(序言|後記)`
	cfg.Recent.SimpleLeadingSpace = "0"

	got := simpleChapterRegexFromUI("第", "混合型數字", "節", `^\s*(序言|後記)`, false)
	if got != cfg.SimpleChapterRegex() {
		t.Fatalf("regex = %q, want config logic %q", got, cfg.SimpleChapterRegex())
	}
}
