package copy

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func getUniqueFileName(targetPath string) string {
	ext := filepath.Ext(targetPath)
	basePath := targetPath[:len(targetPath)-len(ext)]
	date := time.Now().Format("02-01-2006")
	newPath := fmt.Sprintf("%s_%s%s", basePath, date, ext)
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		return newPath
	}

	counter := 1
	for {
		newPath = fmt.Sprintf("%s_%s_%d%s", basePath, date, counter, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

type MediaType int

const (
	Unknown MediaType = iota
	Image
	Video
	Audio
)

func getMediaTypeAndSize(filePath string) (MediaType, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Unknown, 0, fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return Unknown, 0, fmt.Errorf("could not stat file: %w", err)
	}
	if stat.IsDir() {
		return Unknown, 0, fmt.Errorf("file is a directory")
	}
	size := stat.Size()
	if size <= 512 {
		return Unknown, size, nil
	}

	// Read the first 512 bytes to sniff the content type.
	// This is the standard buffer size for http.DetectContentType.
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return Unknown, 0, fmt.Errorf("could not read file: %w", err)
	}

	contentType := http.DetectContentType(buffer[:n])

	if strings.HasPrefix(contentType, "image/") {
		return Image, size, nil
	} else if strings.HasPrefix(contentType, "video/") {
		return Video, size, nil
	} else if strings.HasPrefix(contentType, "audio/") {
		return Audio, size, nil
	}

	return Unknown, size, nil
}

type Source struct {
	Value    string
	FileType MediaType
	Size     int64
	Next     *Source
	Head     *Source
}

type GetFilesListOptions struct {
	DirsToCopy []string
	DirsToSkip []string
	SourcePath string
}

func getFilesList(options GetFilesListOptions) (*Source, int, int64) {
	var currentSource Source
	totalFiles := 0
	var totalBytes int64
	err := filepath.WalkDir(options.SourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}
		dirPathIndex := -1
		for _, dirToCopy := range options.DirsToCopy {
			index := strings.Index(path, dirToCopy)
			if index != -1 {
				dirPathIndex = index + len(dirToCopy)
				break
			}
		}
		if dirPathIndex == -1 {
			return nil
		}
		// Skip only subdirs
		subPath := path[dirPathIndex:]
		for _, dirToSkip := range options.DirsToSkip {
			if strings.Contains(subPath, dirToSkip) {
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		mediaType, size, err := getMediaTypeAndSize(path)
		if err != nil {
			panic(err)
		}
		if mediaType == Unknown {
			return nil
		}
		totalBytes += size
		totalFiles++
		currentSource.Next = &Source{Value: path, Next: nil, Head: currentSource.Head, FileType: mediaType, Size: size}
		if currentSource.Head == nil {
			currentSource.Head = currentSource.Next
		}
		currentSource = *currentSource.Next
		return nil
	})

	if err != nil {
		panic(err)
	}
	return currentSource.Head, totalFiles, totalBytes
}

func copyFile(source Source, targetDir string) {
	filename := filepath.Base(source.Value)
	targetPath := filepath.Join(targetDir, filename)
	_, err := os.Stat(targetPath)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}

	if err == nil {
		targetPath = getUniqueFileName(targetPath)
	}

	sourceFile, err := os.Open(source.Value)
	defer sourceFile.Close()
	if err != nil {
		panic(err)
	}
	targetFile, err := os.Create(targetPath)
	defer targetFile.Close()
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		panic(err)
	}
}

func copyListOfFiles(source *Source, targetDir string, filesCountChan chan int) {
	maxWorkers := 4
	var wg sync.WaitGroup
	currentWorkers := 0
	filesCount := 0
	for source != nil {
		if currentWorkers < maxWorkers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				copyFile(*source, targetDir)
				currentWorkers++
			}()
			currentWorkers++
			source = source.Next
		} else {
			wg.Wait()
			filesCount += maxWorkers
			filesCountChan <- filesCount
			currentWorkers = 0
			continue
		}
	}
	wg.Wait()
	filesCount += currentWorkers
	filesCountChan <- filesCount
	close(filesCountChan)
}

type CopyOptions struct {
	SourcePath string
	TargetDir  string
	DirsToCopy []string
	DirsToSkip []string
	OnProgress func(int, int)
}

func Copy(options CopyOptions) {
	source, filesCount, _ := getFilesList(GetFilesListOptions{
		DirsToCopy: options.DirsToCopy,
		DirsToSkip: options.DirsToSkip,
		SourcePath: options.SourcePath,
	})
	progressChan := make(chan int, 1)
	copyListOfFiles(source, options.TargetDir, progressChan)
	for copiedFiles := range progressChan {
		options.OnProgress(copiedFiles, filesCount)
	}
}
