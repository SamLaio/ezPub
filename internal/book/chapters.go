package book

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"ezpub/internal/lang"
)

type ChapterMark struct {
	StartLine   int
	Level       int
	Title       string
	SkipHeading bool
}

func LoadChapterPlan(path string) ([]ChapterMark, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var marks []ChapterMark
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			return nil, fmt.Errorf("%s:%d: %s", path, lineNo, lang.ErrInvalidChapterPlan)
		}
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || start < 1 {
			return nil, fmt.Errorf("%s:%d: %s", path, lineNo, lang.ErrInvalidChapterPlan)
		}
		level, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || level < 1 {
			return nil, fmt.Errorf("%s:%d: %s", path, lineNo, lang.ErrInvalidChapterPlan)
		}
		skip := false
		if len(parts) >= 4 {
			skip = parseBoolish(parts[3])
		}
		marks = append(marks, ChapterMark{
			StartLine:   start,
			Level:       level,
			Title:       strings.TrimSpace(parts[2]),
			SkipHeading: skip,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return validateChapterPlan(marks)
}

func WriteChapterPlan(path string, marks []ChapterMark) error {
	var b strings.Builder
	b.WriteString("# ezpub chapters v1\n")
	b.WriteString("# start_line\tlevel\ttitle\tskip_heading\n")
	for _, mark := range marks {
		skip := "0"
		if mark.SkipHeading {
			skip = "1"
		}
		b.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\n", mark.StartLine, mark.Level, mark.Title, skip))
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func DetectChapterPlan(text string, opts TextOptions) ([]ChapterMark, error) {
	lines := splitLines(text)
	if opts.SplitCount > 0 {
		return detectEvenChapterPlan(lines, opts.SplitCount), nil
	}
	if opts.FixedChapterLen > 0 {
		return detectFixedChapterPlan(lines, opts.FixedChapterLen, opts.Title), nil
	}
	if strings.TrimSpace(opts.ChapterRegex) == "" {
		return []ChapterMark{{StartLine: 1, Level: 1, Title: defaultPlanTitle(opts.Title)}}, nil
	}
	re, err := regexp.Compile(opts.ChapterRegex)
	if err != nil {
		return nil, err
	}

	var marks []ChapterMark
	prefaceHasText := false
	for i, line := range lines {
		if re.MatchString(line) {
			if len(marks) == 0 && prefaceHasText {
				marks = append(marks, ChapterMark{
					StartLine: 1,
					Level:     1,
					Title:     lang.PrefaceTitle,
				})
			}
			marks = append(marks, ChapterMark{
				StartLine:   i + 1,
				Level:       chapterLevel(line, opts.TOCSpace),
				Title:       strings.TrimSpace(line),
				SkipHeading: true,
			})
			continue
		}
		if len(marks) == 0 && strings.TrimSpace(line) != "" {
			prefaceHasText = true
		}
	}
	if len(marks) == 0 {
		marks = append(marks, ChapterMark{StartLine: 1, Level: 1, Title: defaultPlanTitle(opts.Title)})
	}
	return validateChapterPlan(marks)
}

func SplitChaptersByPlan(text string, marks []ChapterMark, opts TextOptions) ([]Chapter, error) {
	lines := splitLines(text)
	marks, err := validateChapterPlan(marks)
	if err != nil {
		return nil, err
	}

	var chapters []Chapter
	for i, mark := range marks {
		start := mark.StartLine - 1
		if start >= len(lines) {
			break
		}
		contentStart := start
		if mark.SkipHeading {
			contentStart++
		}
		end := len(lines)
		if i+1 < len(marks) {
			end = marks[i+1].StartLine - 1
		}
		if contentStart > end {
			contentStart = end
		}
		title := mark.Title
		if title == "" {
			title = strings.TrimSpace(lines[start])
		}
		if title == "" {
			title = fmt.Sprintf(lang.FixedChapterTitle, i+1)
		}
		chapters = append(chapters, Chapter{
			Title:  title,
			Level:  mark.Level,
			Blocks: paragraphsFromLines(lines[contentStart:end], opts),
		})
	}
	return chapters, nil
}

func SplitChaptersFixed(text string, limit int, opts TextOptions) []Chapter {
	if limit < 1 {
		limit = 1
	}
	var chapters []Chapter
	var buf []string
	count := 0
	flush := func() {
		if len(buf) == 0 {
			return
		}
		chapters = append(chapters, Chapter{
			Title:  fmt.Sprintf(lang.FixedChapterTitle, len(chapters)+1),
			Level:  1,
			Blocks: paragraphsFromLines(buf, opts),
		})
		buf = nil
		count = 0
	}
	for _, line := range splitLines(text) {
		runes := []rune(line)
		for len(runes) > 0 {
			space := limit - count
			if space <= 0 {
				flush()
				space = limit
			}
			if len(runes) <= space {
				buf = append(buf, string(runes))
				count += len(runes)
				runes = nil
				continue
			}
			buf = append(buf, string(runes[:space]))
			runes = runes[space:]
			count += space
			flush()
		}
		if len(runes) == 0 && strings.TrimSpace(line) == "" {
			buf = append(buf, line)
		}
	}
	flush()
	return chapters
}

func SplitChaptersEven(text string, count int, opts TextOptions) []Chapter {
	if count < 1 {
		count = 1
	}
	lines := splitLines(text)
	total := 0
	for _, line := range lines {
		total += len([]rune(line))
	}
	if total == 0 {
		return []Chapter{{
			Title:  defaultPlanTitle(opts.Title),
			Level:  1,
			Blocks: paragraphsFromLines(lines, opts),
		}}
	}

	target := total / count
	if total%count != 0 {
		target++
	}
	var chapters []Chapter
	var buf []string
	size := 0
	for _, line := range lines {
		lineSize := len([]rune(line))
		if len(chapters)+1 < count && len(buf) > 0 && size+lineSize > target {
			chapters = append(chapters, Chapter{
				Title:  fmt.Sprintf(lang.FixedChapterTitle, len(chapters)+1),
				Level:  1,
				Blocks: paragraphsFromLines(buf, opts),
			})
			buf = nil
			size = 0
		}
		buf = append(buf, line)
		size += lineSize
	}
	if len(buf) > 0 || len(chapters) == 0 {
		chapters = append(chapters, Chapter{
			Title:  fmt.Sprintf(lang.FixedChapterTitle, len(chapters)+1),
			Level:  1,
			Blocks: paragraphsFromLines(buf, opts),
		})
	}
	return chapters
}

func detectFixedChapterPlan(lines []string, limit int, title string) []ChapterMark {
	if len(lines) == 0 {
		return []ChapterMark{{StartLine: 1, Level: 1, Title: defaultPlanTitle(title)}}
	}
	var marks []ChapterMark
	count := 0
	for i, line := range lines {
		if len(marks) == 0 || count >= limit {
			marks = append(marks, ChapterMark{
				StartLine: i + 1,
				Level:     1,
				Title:     fmt.Sprintf(lang.FixedChapterTitle, len(marks)+1),
			})
			count = 0
		}
		count += len([]rune(line))
	}
	return marks
}

func detectEvenChapterPlan(lines []string, count int) []ChapterMark {
	if count < 1 {
		count = 1
	}
	total := 0
	for _, line := range lines {
		total += len([]rune(line))
	}
	if total == 0 || len(lines) == 0 {
		return []ChapterMark{{StartLine: 1, Level: 1, Title: fmt.Sprintf(lang.FixedChapterTitle, 1)}}
	}
	target := total / count
	if total%count != 0 {
		target++
	}
	marks := []ChapterMark{{StartLine: 1, Level: 1, Title: fmt.Sprintf(lang.FixedChapterTitle, 1)}}
	size := 0
	for i, line := range lines {
		lineSize := len([]rune(line))
		if len(marks) < count && i > 0 && size+lineSize > target {
			marks = append(marks, ChapterMark{
				StartLine: i + 1,
				Level:     1,
				Title:     fmt.Sprintf(lang.FixedChapterTitle, len(marks)+1),
			})
			size = 0
		}
		size += lineSize
	}
	return marks
}

func validateChapterPlan(marks []ChapterMark) ([]ChapterMark, error) {
	if len(marks) == 0 {
		return nil, fmt.Errorf(lang.ErrEmptyChapterPlan)
	}
	out := append([]ChapterMark(nil), marks...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartLine < out[j].StartLine })
	prev := 0
	for i := range out {
		if out[i].StartLine < 1 || out[i].StartLine <= prev {
			return nil, fmt.Errorf(lang.ErrInvalidChapterPlan)
		}
		if out[i].Level < 1 {
			out[i].Level = 1
		}
		if out[i].Level > 6 {
			out[i].Level = 6
		}
		prev = out[i].StartLine
	}
	return out, nil
}

func defaultPlanTitle(title string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	return lang.DefaultBookTitle
}

func parseBoolish(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}
