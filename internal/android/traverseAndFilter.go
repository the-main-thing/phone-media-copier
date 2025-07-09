package android

import (
	"bufio"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func traverseAndFilter(adbPath string, currentDeviceDir string, fileChan chan<- string, semLs chan struct{}, wgTraversal *sync.WaitGroup) {
	cleanedCurrentDir := filepath.Clean(currentDeviceDir)
	if cleanedCurrentDir == "." {
		cleanedCurrentDir = "/"
	}
	cleanedCurrentDir = strings.TrimSuffix(cleanedCurrentDir, "/") + "/"

	visitedDirsMutex.Lock()
	if visitedDirs[cleanedCurrentDir] {
		visitedDirsMutex.Unlock()
		return
	}
	visitedDirs[cleanedCurrentDir] = true
	visitedDirsMutex.Unlock()

	cmd := exec.Command(adbPath, "shell", "ls", "-F", cleanedCurrentDir)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe for 'ls -F %s': %v", cleanedCurrentDir, err)
		return
	}
	defer stdout.Close()

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting 'ls -F %s': %v (Stderr: %s)", cleanedCurrentDir, err, stderr.String())
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "." || line == ".." { // Skip empty lines, current dir, parent dir
			continue
		}

		fullPath := filepath.ToSlash(filepath.Join(cleanedCurrentDir, line))
		fullPath = strings.TrimSuffix(fullPath, "//")

		if strings.HasSuffix(line, "/") {
			// Skip well-known problematic, large, or irrelevant system directories to avoid errors and unnecessary traversal
			if strings.HasPrefix(fullPath, "/data/") ||
				strings.HasPrefix(fullPath, "/system/") ||
				strings.HasPrefix(fullPath, "/proc/") ||
				strings.HasPrefix(fullPath, "/dev/") ||
				strings.HasPrefix(fullPath, "/acct/") ||
				strings.HasPrefix(fullPath, "/sys/") ||
				strings.Contains(fullPath, "/Android/data/") || // App private data
				strings.Contains(fullPath, "/Android/obb/") || // OBB files, large but rarely media
				strings.Contains(fullPath, "/lost+found/") {
				continue
			}

			// Recurse into subdirectory
			wgTraversal.Add(1)
			semLs <- struct{}{} // Acquire a slot for the recursive 'ls'
			go func(dir string) {
				defer wgTraversal.Done()
				defer func() { <-semLs }() // Release the slot
				traverseAndFilter(adbPath, dir, fileChan, semLs, wgTraversal)
			}(fullPath)
		} else {
			if passesFilter(fullPath) {
				fileChan <- fullPath
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading output from 'ls -F %s': %v", cleanedCurrentDir, err)
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("'ls -F %s' command failed: %v (Stderr: %s)", cleanedCurrentDir, err, stderr.String())
	}
}

func passesFilter(filePath string) bool {
	filePathLower := strings.ToLower(filePath)
	ext := strings.ToLower(filepath.Ext(filePathLower))

	if ext == ".gif" {
		return false
	}

	// 1. Common Camera Photos (often in DCIM/Camera)
	if strings.Contains(filePathLower, "/dcim/camera/") && (ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".heic") {
		return true
	}

	// 2. Telegram Media (various subdirectories under /Telegram/)
	if strings.Contains(filePathLower, "/telegram/") {
		if strings.Contains(filePathLower, "/telegram images/") && (ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp") {
			return true
		}
		if strings.Contains(filePathLower, "/telegram video/") && (ext == ".mp4" || ext == ".mov" || ext == ".webm" || ext == ".avi") {
			return true
		}
		if strings.Contains(filePathLower, "/telegram audio/") && (ext == ".mp3" || ext == ".ogg" || ext == ".aac" || ext == ".m4a") {
			return true
		}
		// "Documents" folders may contain various files, only allow media
		if strings.Contains(filePathLower, "/telegram documents/") && isMediaExtension(ext) {
			return true
		}
	}

	// 3. WhatsApp Media (various subdirectories under /WhatsApp/Media/)
	if strings.Contains(filePathLower, "/whatsapp/media/") {
		if strings.Contains(filePathLower, "/whatsapp images/") && (ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp") {
			return true
		}
		if strings.Contains(filePathLower, "/whatsapp video/") && (ext == ".mp4" || ext == ".mov" || ext == ".3gp") {
			return true
		}
		if strings.Contains(filePathLower, "/whatsapp audio/") && (ext == ".ogg" || ext == ".opus" || ext == ".aac" || ext == ".m4a") {
			return true
		}
		if strings.Contains(filePathLower, "/whatsapp voice notes/") && (ext == ".opus" || ext == ".aac" || ext == ".m4a") {
			return true
		}
		// "Documents" folders may contain various files, only allow media
		if strings.Contains(filePathLower, "/whatsapp documents/") && isMediaExtension(ext) {
			return true
		}
	}

	// 4. General Media: Also consider media files found in other common user directories
	if isMediaExtension(ext) {
		if !strings.HasPrefix(filePathLower, "/data/") &&
			!strings.HasPrefix(filePathLower, "/system/") &&
			!strings.Contains(filePathLower, "/android/data/") &&
			!strings.Contains(filePathLower, "/android/obb/") {
			return true
		}
	}

	return false
}
