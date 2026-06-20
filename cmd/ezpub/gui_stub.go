//go:build !windows

package main

import "errors"

func runGUI(args []string) error {
	return errors.New("GUI is only available on Windows")
}
