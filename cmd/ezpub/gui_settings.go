//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const guiSettingsName = "gui.json"

type guiSettings struct {
	FontType     int      `json:"fontType"`
	CustomFont   string   `json:"customFont"`
	EmbeddedFont string   `json:"embeddedFont"`
	SubsetFonts  bool     `json:"subsetFonts"`
	Subject      string   `json:"subject"`
	SubjectTags  []string `json:"subjectTags"`
	Language     string   `json:"language"`
	CSSMode      int      `json:"cssMode"`
	VerticalCSS  bool     `json:"verticalCSS"`
	SaveCSS      bool     `json:"saveCSS"`
}

func loadGUISettings() (guiSettings, bool) {
	for _, path := range settingReadPaths(guiSettingsName) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var settings guiSettings
		if err := json.Unmarshal(data, &settings); err == nil {
			return settings, true
		}
	}
	return guiSettings{}, false
}

func saveGUISettings(settings guiSettings) error {
	path, err := settingWritePath(guiSettingsName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func validGUIFontType(value int) bool {
	return value >= 1 && value <= 3
}

func validGUICSSMode(value int) bool {
	return value >= 0 && value <= 2
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
