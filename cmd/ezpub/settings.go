package main

import (
	"os"
	"path/filepath"
	"strings"
)

func settingReadPaths(name string) []string {
	var paths []string
	add := func(path string) {
		path = filepath.Clean(path)
		for _, existing := range paths {
			if strings.EqualFold(existing, path) {
				return
			}
		}
		paths = append(paths, path)
	}
	if exe, err := os.Executable(); err == nil {
		add(filepath.Join(filepath.Dir(exe), "setting", name))
	}
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "setting", name))
	}
	return paths
}

func settingWritePath(name string) (string, error) {
	for _, path := range settingReadPaths(name) {
		if info, err := os.Stat(filepath.Dir(path)); err == nil && info.IsDir() {
			return path, nil
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, "setting", name), nil
}

func ensureSettingFile(name string) (string, error) {
	path, err := settingWritePath(name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_CREATE, 0644)
	if err != nil {
		return "", err
	}
	return path, file.Close()
}

func readSettingText(name string) string {
	for _, path := range settingReadPaths(name) {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(strings.TrimPrefix(string(data), "\ufeff"))
		}
	}
	return ""
}

func writeSettingText(name, value string) error {
	path, err := settingWritePath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(value)), 0644)
}
