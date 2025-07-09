package android

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type ADB struct {
	Path string
}

func (c *ADB) Cleanup() {
	os.RemoveAll(filepath.Dir(c.Path))
}

//go:embed adb/*
var adbFiles embed.FS

func extractADB() (ADB, error) {
	adbName := "adb"
	if runtime.GOOS == "windows" {
		adbName = "adb.exe"
	}

	tempDir, err := os.MkdirTemp("", "adb-tmp")
	if err != nil {
		return ADB{}, fmt.Errorf("failed to create temp dir: %v", err)
	}

	adbPath := filepath.Join(tempDir, adbName)
	adbData, err := adbFiles.ReadFile(fmt.Sprintf("adb/%s", adbName))
	if err != nil {
		return ADB{}, fmt.Errorf("failed to read embedded ADB: %v", err)
	}

	if err := os.WriteFile(adbPath, adbData, 0755); err != nil {
		return ADB{}, fmt.Errorf("failed to write ADB binary: %v", err)
	}

	err = checkAdbConnection(adbPath)
	if err != nil {
		return ADB{}, err
	}

	return ADB{Path: adbPath}, nil
}

func checkAdbConnection(adbPath string) error {
	cmd := exec.Command(adbPath, "devices")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if strings.Contains(line, "device") && !strings.Contains(line, "List of devices attached") {
			return nil
		}
	}
	return errors.New("no device connected")
}
