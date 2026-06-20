package main

import (
	"os"
	"path/filepath"
	"strings"
)

const cachedCSSName = "custom.css"

func loadCachedCSS(enabled bool) (string, error) {
	if !enabled {
		return "", nil
	}
	path, err := cachedCSSPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func saveCachedCSS(enabled bool, css string) error {
	if !enabled || strings.TrimSpace(css) == "" {
		return nil
	}
	return writeCachedCSS(css)
}

func writeCachedCSS(css string) error {
	path, err := cachedCSSPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(css), 0644)
}

func cachedCSSPath() (string, error) {
	return settingWritePath(cachedCSSName)
}
