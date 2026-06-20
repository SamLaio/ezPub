package convert

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"ezpub/internal/lang"
)

func SubsetFonts(fonts []string, text, tool string) ([]string, func(), error) {
	if len(fonts) == 0 {
		return fonts, func() {}, nil
	}
	found, ok := FindFontSubsetter(tool)
	if !ok {
		return nil, func() {}, fmt.Errorf(lang.ErrFontSubsetterNeeded)
	}
	tool = found

	dir, err := os.MkdirTemp("", "ezpub-font-subset-*")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	textPath := filepath.Join(dir, "text.txt")
	if err := os.WriteFile(textPath, []byte(subsetCharacters(text)), 0644); err != nil {
		cleanup()
		return nil, func() {}, err
	}

	out := make([]string, 0, len(fonts))
	for i, font := range fonts {
		ext := filepath.Ext(font)
		name := strings.TrimSuffix(filepath.Base(font), ext)
		subsetPath := filepath.Join(dir, fmt.Sprintf("%s-subset-%d%s", name, i+1, ext))
		cmd := exec.Command(tool,
			font,
			"--text-file="+textPath,
			"--output-file="+subsetPath,
			"--layout-features=*",
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf(lang.ErrFontSubsetFailed, err, stderr.String())
		}
		usePath, err := smallerFontPath(font, subsetPath)
		if err != nil {
			cleanup()
			return nil, func() {}, err
		}
		out = append(out, usePath)
	}
	return out, cleanup, nil
}

func FindFontSubsetter(tool string) (string, bool) {
	if path := strings.TrimSpace(tool); path != "" {
		return existingFile(path)
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "tools", "pyftsubset.exe"),
			filepath.Join(dir, "pyftsubset.exe"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "release", "tools", "pyftsubset.exe"),
			filepath.Join(wd, "release", "pyftsubset.exe"),
			filepath.Join(wd, "tools", "pyftsubset.exe"),
			filepath.Join(wd, "pyftsubset.exe"),
		)
	}
	for _, candidate := range candidates {
		if path, ok := existingFile(candidate); ok {
			return path, true
		}
	}
	if found, err := exec.LookPath("pyftsubset.exe"); err == nil {
		return found, true
	}
	if found, err := exec.LookPath("pyftsubset"); err == nil {
		return found, true
	}
	return "", false
}

func existingFile(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

func subsetCharacters(text string) string {
	seen := map[rune]bool{}
	for _, r := range text {
		if unicode.IsSpace(r) {
			r = ' '
		}
		seen[r] = true
	}
	runes := make([]rune, 0, len(seen))
	for r := range seen {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
	return string(runes)
}

func smallerFontPath(originalPath, subsetPath string) (string, error) {
	original, err := os.Stat(originalPath)
	if err != nil {
		return "", err
	}
	subset, err := os.Stat(subsetPath)
	if err != nil {
		return "", err
	}
	if subset.Size() < original.Size() {
		return subsetPath, nil
	}
	_ = os.Remove(subsetPath)
	return originalPath, nil
}
