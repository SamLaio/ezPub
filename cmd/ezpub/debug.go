package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"
)

var (
	debugLogEnabled = false
	debugMu         sync.Mutex
	debugLogger     *log.Logger
	debugFile       *os.File
)

func initDebugLog(enabled bool) error {
	debugLogEnabled = enabled
	if !debugLogEnabled {
		return nil
	}

	dir, err := executableLogDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, time.Now().Format("20060102-150405")+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	debugMu.Lock()
	if debugFile != nil {
		_ = debugFile.Close()
	}
	debugFile = file
	debugLogger = log.New(file, "", log.LstdFlags|log.Lmicroseconds)
	debugMu.Unlock()

	debugLog("debug log started: %s", path)
	return nil
}

func closeDebugLog() {
	debugMu.Lock()
	defer debugMu.Unlock()
	if debugFile != nil {
		debugLogger.Printf("debug log closed")
		_ = debugFile.Close()
		debugFile = nil
		debugLogger = nil
	}
}

func debugLog(format string, args ...interface{}) {
	if !debugLogEnabled {
		return
	}
	debugMu.Lock()
	defer debugMu.Unlock()
	if debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}

func debugRecover(scope string) {
	if r := recover(); r != nil {
		debugLog("panic in %s: %v\n%s", scope, r, debug.Stack())
	}
}

func executableLogDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), "log"), nil
}
