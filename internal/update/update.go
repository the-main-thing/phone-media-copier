package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const RELEASES_URL = "https://api.github.com/repos/the-main-thing/phone-media-copier/releases/latest"
const WINDOWS_BINARY_NAME = "phone-media-copier-windows-amd64.exe"
const LINUX_BINARY_NAME = "phone-media-copier-linux-amd64"
const MAC_ARM64_BINARY_NAME = "phone-media-copier-darwin-arm64"
const MAC_AMD64_BINARY_NAME = "phone-media-copier-darwin-amd64"

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func getBinaryName() string {
	if runtime.GOOS == "windows" {
		return WINDOWS_BINARY_NAME
	}
	if runtime.GOOS == "linux" {
		return LINUX_BINARY_NAME
	}
	if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "arm64" {
			return MAC_ARM64_BINARY_NAME
		}
		return MAC_AMD64_BINARY_NAME
	}
	return ""
}

func getNewBinaryUrl() (string, error) {
	resp, err := http.Get(RELEASES_URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("could not get latest version: %d", resp.StatusCode)
	}
	var release Release
	error := json.NewDecoder(resp.Body).Decode(&release)
	if error != nil {
		return "", error
	}
	if release.TagName == VERSION {
		return "", nil
	}
	var binaryUrl string
	binaryName := getBinaryName()
	if binaryName == "" {
		return "", errors.New("unsupported platform")
	}

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, binaryName) {
			binaryUrl = asset.URL
			break
		}
	}

	return binaryUrl, nil
}

func getCurrentBinaryPath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	mode := info.Mode()
	if mode.Type() == os.ModeSymlink {
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return "", err
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			cwd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			path = filepath.Join(cwd, path)
		}
		path = absPath
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("binary exect path: could not get absolute path from symlink. there is relative path somewhere")
	}
	path = absPath
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if stat.IsDir() {
		return "", errors.New("binary exec path symlink points to a directory")
	}
	return path, nil
}

func Update() (bool, error) {
	binaryPath, err := getCurrentBinaryPath()
	if err != nil {
		return false, err
	}

	binaryUrl, err := getNewBinaryUrl()
	if err != nil {
		return false, err
	}
	if binaryUrl == "" {
		return false, nil
	}

	backupPath := binaryPath + ".bak"
	err = os.Rename(binaryPath, backupPath)
	if err != nil {
		return false, err
	}

	newFile, err := os.Create(binaryPath)
	defer newFile.Close()
	if err != nil {
		return false, err
	}

	resp, err := http.Get(binaryUrl)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	_, err = io.Copy(newFile, resp.Body)
	if err != nil {
		return false, err
	}
	err = newFile.Close()
	if err != nil {
		return false, err
	}

	err = os.Remove(backupPath)
	if err != nil {
		return false, err
	}

	return true, nil

}
