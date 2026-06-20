//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var shellExecuteW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")

func openSystemFile(path string) error {
	operation, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	ret, _, callErr := shellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(operation)),
		uintptr(unsafe.Pointer(file)),
		0,
		0,
		1, // SW_SHOWNORMAL
	)
	if ret <= 32 {
		if callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return fmt.Errorf("ShellExecute failed with code %d", ret)
	}
	return nil
}
