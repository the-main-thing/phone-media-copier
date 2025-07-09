package android

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeImage
	FileTypeVideo
	FileTypeAudio
)

const MAX_CONCURRENT_PULLS = 5

func getFileSize(adbPath string, filePath string) (int64, error) {
	cmd := exec.Command(adbPath, "shell", "stat", "-c", "%s", filePath)

	stdout, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	size, err := strconv.ParseInt(strings.TrimSpace(string(stdout)), 10, 64)
	if err != nil {
		return 0, err
	}

	return size, nil
}

func pullFile(adbPath string, sourcePath string, targetDir string) error {
	targetPath := getFilePath(sourcePath, targetDir, "")
	stat, err := os.Stat(targetPath)
	if err == nil {
		fileSize, err := getFileSize(adbPath, sourcePath)
		if err != nil {
			return err
		}
		if stat.Size() == fileSize {
			return nil
		}
		// same name, different size
		index := 1
		for {
			targetPath = getFilePath(sourcePath, targetDir, strconv.Itoa(index))
			stat, err := os.Stat(targetPath)
			if err == nil {
				if stat.Size() == fileSize {
					return nil
				}
				index++
				continue
			}
			if os.IsNotExist(err) {
				break
			}
			return err
		}
	}

	if !os.IsNotExist(err) {
		return err
	}

	return exec.Command(adbPath, "pull", sourcePath, targetPath).Run()
}

func Copy(targetDir string, onProgress func(progress int)) error {
	adb, err := extractADB()
	defer adb.Cleanup()
	if err != nil {
		return err
	}

	err = checkAdbConnection(adb.Path)
	if err != nil {
		return err
	}

	filesToCopy, totalFiles, err := getFilesPathsToCopy(adb.Path)
	if err != nil {
		return err
	}

	var (
		processedCount atomic.Int64
		errGroup       sync.WaitGroup
		tasksChan      = make(chan string, MAX_CONCURRENT_PULLS)
		errorsChan     = make(chan error, MAX_CONCURRENT_PULLS)
	)

	errGroup.Add(MAX_CONCURRENT_PULLS)
	for range MAX_CONCURRENT_PULLS {
		go func() {
			defer errGroup.Done()
			for file := range tasksChan {
				if err := pullFile(adb.Path, file, targetDir); err != nil {
					errorsChan <- fmt.Errorf("failed to copy %s: %w", file, err)
					continue
				}

				count := processedCount.Add(1)
				progress := int(float64(count) / float64(totalFiles) * 100)
				onProgress(progress)
			}
		}()
	}

	go func() {
		defer close(tasksChan)
		for current := filesToCopy.Head; current != nil; current = current.Next {
			if current.Path == "" {
				continue
			}
			tasksChan <- current.Path
		}
	}()

	go func() {
		errGroup.Wait()
		close(errorsChan)
	}()

	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple copy errors occurred: %v", errors)
	}

	return nil
}
