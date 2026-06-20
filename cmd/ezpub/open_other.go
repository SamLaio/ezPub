//go:build !windows

package main

import (
	"os/exec"
	"runtime"
)

func openSystemFile(path string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", path).Start()
	}
	return exec.Command("xdg-open", path).Start()
}
