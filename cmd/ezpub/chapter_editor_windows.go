//go:build windows

package main

import (
	"fmt"
	"sort"
	"strings"

	"ezpub/internal/book"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type chapterEditorRow struct {
	Index       int
	Title       string
	Line        int
	Level       int
	SkipHeading bool
	Deleted     bool
}

type chapterEditorModel struct {
	walk.TableModelBase
	rows []chapterEditorRow
}

func (m *chapterEditorModel) RowCount() int {
	return len(m.rows)
}

func (m *chapterEditorModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.rows) {
		return ""
	}
	item := m.rows[row]
	switch col {
	case 0:
		if item.Level > 1 {
			return strings.Repeat("+", item.Level-1) + fmt.Sprint(item.Index)
		}
		return fmt.Sprint(item.Index)
	case 1:
		title := item.Title
		if item.Deleted {
			return "[刪除] " + title
		}
		return title
	case 2:
		return item.Line
	default:
		return ""
	}
}

func (m *chapterEditorModel) renumber() {
	sort.SliceStable(m.rows, func(i, j int) bool {
		return m.rows[i].Line < m.rows[j].Line
	})
	for i := range m.rows {
		m.rows[i].Index = i + 1
		if m.rows[i].Level < 1 {
			m.rows[i].Level = 1
		}
		if m.rows[i].Title == "" {
			m.rows[i].Title = fmt.Sprintf("第%d章", i+1)
		}
		if m.rows[i].Line < 1 {
			m.rows[i].Line = 1
		}
	}
	m.PublishRowsReset()
}

func (m *chapterEditorModel) marks() []book.ChapterMark {
	out := make([]book.ChapterMark, 0, len(m.rows))
	for _, row := range m.rows {
		if row.Deleted {
			continue
		}
		out = append(out, book.ChapterMark{
			StartLine:   row.Line,
			Level:       maxInt(1, row.Level),
			Title:       strings.TrimSpace(row.Title),
			SkipHeading: row.SkipHeading,
		})
	}
	return out
}

func showChapterEditor(owner walk.Form, title, status, outPath string, marks []book.ChapterMark, sourceLines []string) ([]book.ChapterMark, bool) {
	defer debugRecover("showChapterEditor")
	debugLog("show chapter editor title=%q status=%q out=%q marks=%d sourceLines=%d", title, status, outPath, len(marks), len(sourceLines))

	rows := make([]chapterEditorRow, 0, len(marks))
	for i, mark := range marks {
		rows = append(rows, chapterEditorRow{
			Index:       i + 1,
			Title:       mark.Title,
			Line:        mark.StartLine,
			Level:       maxInt(1, mark.Level),
			SkipHeading: mark.SkipHeading,
		})
	}
	if len(rows) == 0 {
		rows = append(rows, chapterEditorRow{Index: 1, Title: "第一章", Line: 1, Level: 1})
	}
	model := &chapterEditorModel{rows: rows}
	model.renumber()

	var accepted bool
	var dlg *walk.Dialog
	var tv *walk.TableView
	var maxLineNE, addLineNE *walk.NumberEdit
	selectedRows := func() []int {
		if tv == nil {
			return nil
		}
		return validChapterSelection(tv.SelectedIndexes(), tv.CurrentIndex(), len(model.rows))
	}
	toggleSelected := func() {
		rows := selectedRows()
		if len(rows) == 0 {
			return
		}
		for _, idx := range rows {
			model.rows[idx].Deleted = !model.rows[idx].Deleted
			model.PublishRowChanged(idx)
			debugLog("chapter editor toggle delete row=%d deleted=%v", idx, model.rows[idx].Deleted)
		}
	}
	removeSelected := func() {
		rows := selectedRows()
		if len(rows) == 0 {
			return
		}
		for i := len(rows) - 1; i >= 0; i-- {
			idx := rows[i]
			debugLog("chapter editor remove row=%d", idx)
			model.rows = append(model.rows[:idx], model.rows[idx+1:]...)
		}
		model.renumber()
		idx := rows[0]
		if idx > len(model.rows)-1 {
			idx = len(model.rows) - 1
		}
		if idx >= 0 {
			_ = tv.SetCurrentIndex(idx)
		}
	}
	adjustLevel := func(delta int) {
		rows := selectedRows()
		if len(rows) == 0 {
			return
		}
		changed := adjustChapterLevels(model, rows, delta)
		_ = tv.SetFocus()
		debugLog("chapter editor adjust level selected=%d changed=%d", len(rows), changed)
	}
	addChapter := func() {
		line := numberEditInt(addLineNE, 1)
		idx, added, err := addChapterBySourceLine(model, sourceLines, line)
		if err != nil {
			walk.MsgBox(dlg, "ezPub", err.Error(), walk.MsgBoxIconError)
			debugLog("chapter editor add line failed line=%d sourceLines=%d err=%v", line, len(sourceLines), err)
			return
		}
		if idx >= 0 {
			_ = tv.SetCurrentIndex(idx)
			tv.EnsureItemVisible(idx)
		}
		if added {
			debugLog("chapter editor add line=%d row=%d", line, idx)
		} else {
			debugLog("chapter editor add existing line=%d row=%d", line, idx)
		}
	}
	markShortChapters := func() {
		limit := numberEditInt(maxLineNE, 1)
		if limit < 1 {
			limit = 1
		}
		changed := 0
		for i := range model.rows {
			nextLine := model.rows[i].Line + limit + 1
			if i+1 < len(model.rows) {
				nextLine = model.rows[i+1].Line
			}
			if nextLine-model.rows[i].Line <= limit {
				model.rows[i].Deleted = true
				changed++
			}
		}
		model.PublishRowsReset()
		debugLog("chapter editor mark short limit=%d changed=%d", limit, changed)
	}
	editCurrent := func() {
		if tv == nil {
			return
		}
		idx := tv.CurrentIndex()
		if idx < 0 || idx >= len(model.rows) {
			return
		}
		if value, ok := editChapterTitleDialog(owner, model.rows[idx].Title); ok {
			model.rows[idx].Title = value
			model.PublishRowChanged(idx)
			debugLog("chapter editor edit title row=%d title=%q", idx, value)
		}
	}
	var titleLabel, statusLabel, splitLabel, shortLabel, newLineLabel, helpLabel, listLabel *walk.Label
	var markShortCB *walk.CheckBox
	var deleteBtn, addBtn, saveBtn, cancelBtn *walk.PushButton
	dialog := Dialog{
		AssignTo: &dlg,
		Title:    "章節編輯  - ezPub Sam版",
		Size:     Size{Width: 760, Height: 540},
		MinSize:  Size{Width: 760, Height: 540},
		MaxSize:  Size{Width: 760, Height: 540},
		Layout:   Flow{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			fixedComposite{
				MinSize: Size{Width: 740, Height: 500},
				MaxSize: Size{Width: 740, Height: 500},
				Children: []Widget{
					Label{AssignTo: &titleLabel, Text: title},
					Label{AssignTo: &statusLabel, Text: status, TextColor: walk.RGB(160, 0, 0)},
					Label{AssignTo: &splitLabel, Text: "分割結果："},
					CheckBox{AssignTo: &markShortCB, Text: "標記不超過"},
					NumberEdit{AssignTo: &maxLineNE, Decimals: 0, Increment: 1, SpinButtonsVisible: true},
					Label{AssignTo: &shortLabel, Text: "行的章節"},
					PushButton{AssignTo: &deleteBtn, Text: "點擊刪除", OnClicked: func() {
						markShortChapters()
					}},
					Label{AssignTo: &newLineLabel, Text: "新章節，行號："},
					NumberEdit{AssignTo: &addLineNE, Decimals: 0, Increment: 1, SpinButtonsVisible: true},
					PushButton{AssignTo: &addBtn, Text: "點擊添加", OnClicked: addChapter},
					Label{AssignTo: &helpLabel, Text: "Tab/Shift Tab 調整層級；Enter: 刪除/恢復選中的目錄；Delete/Shift+Delete：刪除/徹底刪除選中的目錄"},
					Label{AssignTo: &listLabel, Text: "章節列表"},
					TableView{
						AssignTo:                 &tv,
						Model:                    model,
						AlternatingRowBG:         true,
						ColumnsOrderable:         false,
						MultiSelection:           true,
						NotSortableByHeaderClick: true,
						LastColumnStretched:      false,
						OnItemActivated:          editCurrent,
						OnKeyDown: func(key walk.Key) {
							switch key {
							case walk.KeyReturn:
								toggleSelected()
							case walk.KeyDelete:
								if walk.ShiftDown() {
									removeSelected()
								} else {
									toggleSelected()
								}
							case walk.KeyTab:
								if walk.ShiftDown() {
									adjustLevel(-1)
								} else {
									adjustLevel(1)
								}
							}
						},
						Columns: []TableViewColumn{
							{Title: "序号", Width: 90},
							{Title: "章節名稱（雙擊編輯）", Width: 470},
							{Title: "行号", Width: 80},
						},
					},
					PushButton{AssignTo: &saveBtn, Text: "保存修改", OnClicked: func() {
						debugLog("chapter editor save out=%q rows=%d", outPath, len(model.rows))
						if len(model.marks()) == 0 {
							walk.MsgBox(dlg, "ezPub", "章節列表不可為空", walk.MsgBoxIconError)
							return
						}
						accepted = true
						dlg.Accept()
					}},
					PushButton{AssignTo: &cancelBtn, Text: "取消修改", OnClicked: func() {
						debugLog("chapter editor cancel")
						dlg.Cancel()
					}},
				},
				Place: func() {
					setWidgetBounds(titleLabel, 14, 10, 710, 24)
					setWidgetBounds(statusLabel, 14, 36, 710, 24)
					setWidgetBounds(splitLabel, 14, 68, 80, 24)
					setWidgetBounds(markShortCB, 110, 68, 90, 24)
					setWidgetBounds(maxLineNE, 200, 66, 70, 26)
					setWidgetBounds(shortLabel, 276, 68, 80, 24)
					setWidgetBounds(deleteBtn, 366, 64, 90, 30)
					setWidgetBounds(newLineLabel, 488, 68, 100, 24)
					setWidgetBounds(addLineNE, 585, 66, 55, 26)
					setWidgetBounds(addBtn, 656, 64, 70, 30)
					setWidgetBounds(helpLabel, 14, 100, 720, 24)
					setWidgetBounds(listLabel, 14, 128, 80, 24)
					setWidgetBounds(tv, 14, 154, 720, 280)
					setWidgetBounds(saveBtn, 205, 448, 100, 30)
					setWidgetBounds(cancelBtn, 395, 448, 100, 30)
				},
			},
		},
	}
	if err := dialog.Create(owner); err != nil {
		debugLog("chapter editor error: %v", err)
		walk.MsgBox(owner, "ezPub", fmt.Sprintf("章節編輯器開啟失敗：%v", err), walk.MsgBoxIconError)
		return nil, false
	}
	if err := installChapterEditorShortcuts(dlg, adjustLevel); err != nil {
		debugLog("chapter editor shortcut error: %v", err)
	}
	setDialogNumberValue := func(ne *walk.NumberEdit, min, max, value float64) {
		if ne == nil {
			return
		}
		_ = ne.SetRange(min, max)
		_ = ne.SetValue(value)
	}
	setDialogNumberValue(maxLineNE, 1, 999999, 1)
	setDialogNumberValue(addLineNE, 1, float64(maxInt(1, len(sourceLines))), 1)
	dlg.Run()
	if !accepted {
		return nil, false
	}
	return model.marks(), true
}

func editChapterTitleDialog(owner walk.Form, value string) (string, bool) {
	var dlg *walk.Dialog
	var titleLE *walk.LineEdit
	var okBtn, cancelBtn *walk.PushButton
	var accepted bool
	_, err := Dialog{
		AssignTo: &dlg,
		Title:    "編輯章節名稱",
		Size:     Size{Width: 360, Height: 120},
		Layout:   Flow{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			fixedComposite{
				MinSize: Size{Width: 340, Height: 82},
				MaxSize: Size{Width: 340, Height: 82},
				Children: []Widget{
					LineEdit{AssignTo: &titleLE, Text: value},
					PushButton{AssignTo: &okBtn, Text: "確定", OnClicked: func() {
						if strings.TrimSpace(titleLE.Text()) == "" {
							walk.MsgBox(dlg, "ezPub", "章節名稱不可為空", walk.MsgBoxIconError)
							return
						}
						accepted = true
						dlg.Accept()
					}},
					PushButton{AssignTo: &cancelBtn, Text: "取消", OnClicked: func() { dlg.Cancel() }},
				},
				Place: func() {
					setWidgetBounds(titleLE, 12, 10, 310, 24)
					setWidgetBounds(okBtn, 156, 46, 76, 28)
					setWidgetBounds(cancelBtn, 244, 46, 76, 28)
				},
			},
		},
	}.Run(owner)
	if err != nil || !accepted {
		return "", false
	}
	return strings.TrimSpace(titleLE.Text()), true
}

func installChapterEditorShortcuts(dlg *walk.Dialog, adjustLevel func(int)) error {
	if dlg == nil || adjustLevel == nil {
		return nil
	}
	add := func(mods walk.Modifiers, delta int) error {
		action := walk.NewAction()
		if err := action.SetShortcut(walk.Shortcut{Modifiers: mods, Key: walk.KeyTab}); err != nil {
			return err
		}
		action.Triggered().Attach(func() {
			adjustLevel(delta)
		})
		return dlg.ShortcutActions().Add(action)
	}
	if err := add(0, 1); err != nil {
		return err
	}
	return add(walk.ModShift, -1)
}

func adjustChapterLevel(model *chapterEditorModel, row, delta int) bool {
	if model == nil || row < 0 || row >= len(model.rows) {
		return false
	}
	level := model.rows[row].Level + delta
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	if level == model.rows[row].Level {
		return false
	}
	model.rows[row].Level = level
	model.PublishRowChanged(row)
	return true
}

func adjustChapterLevels(model *chapterEditorModel, rows []int, delta int) int {
	changed := 0
	for _, row := range rows {
		if adjustChapterLevel(model, row, delta) {
			changed++
		}
	}
	return changed
}

func validChapterSelection(selected []int, current, rowCount int) []int {
	seen := make(map[int]bool, len(selected)+1)
	rows := make([]int, 0, len(selected)+1)
	add := func(idx int) {
		if idx < 0 || idx >= rowCount || seen[idx] {
			return
		}
		seen[idx] = true
		rows = append(rows, idx)
	}
	for _, idx := range selected {
		add(idx)
	}
	if len(rows) == 0 {
		add(current)
	}
	sort.Ints(rows)
	return rows
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func numberEditInt(ne *walk.NumberEdit, fallback int) int {
	if ne == nil {
		return fallback
	}
	value := int(ne.Value())
	if value == 0 {
		return fallback
	}
	return value
}

func addChapterBySourceLine(model *chapterEditorModel, sourceLines []string, line int) (int, bool, error) {
	if model == nil {
		return -1, false, fmt.Errorf("章節列表未初始化")
	}
	if line < 1 {
		line = 1
	}
	if line > len(sourceLines) {
		return -1, false, fmt.Errorf("行號超出原始 TXT 行數：%d / %d", line, len(sourceLines))
	}
	for i := range model.rows {
		if model.rows[i].Line == line {
			model.rows[i].Deleted = false
			model.PublishRowChanged(i)
			return i, false, nil
		}
	}
	title := strings.TrimSpace(sourceLines[line-1])
	if title == "" {
		title = fmt.Sprintf("第%d行", line)
	}
	model.rows = append(model.rows, chapterEditorRow{
		Title:       title,
		Line:        line,
		Level:       1,
		SkipHeading: true,
	})
	model.renumber()
	for i := range model.rows {
		if model.rows[i].Line == line {
			return i, true, nil
		}
	}
	return -1, true, nil
}
