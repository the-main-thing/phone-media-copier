package android

import (
	"path/filepath"
	"strings"
	"sync"
)

var deviceBaseDirs = []string{
	"/sdcard/",               // The most common and canonical internal storage root (often a symlink)
	"/storage/emulated/0/",   // Modern Android's primary user storage (often symlinked to /sdcard/)
	"/mnt/sdcard/",           // Older Android versions or some custom ROMs
	"/storage/self/primary/", // Less common, but sometimes seen (often symlinked to /sdcard/ or /storage/emulated/0/)
	"/data/media/0/",         // Lower-level path for internal storage, may require root or specific permissions
	"/storage/",              // General mount point for external storage (e.g., /storage/XXXX-XXXX/)
	"/mnt/extsd/",            // Common for external SD cards
	"/mnt/external_sd/",      // Another variation for external SD cards
	"/Removable/MicroSD/",    // Some device manufacturers use specific paths
	"/storage/extSdCard/",    // Samsung devices
	"/mnt/media_rw/sdcard1/", // Less common direct access path to raw SD card
}

const MAX_CONCURRENT_LS = 3

var visitedDirs = make(map[string]bool)
var visitedDirsMutex sync.Mutex

type FilePath struct {
	Path string
	Head *FilePath
	Next *FilePath
}

func getFilesPathsToCopy(adbPath string) (FilePath, int, error) {
	var filesToCopy FilePath
	size := 0

	var wgTraversal sync.WaitGroup
	semLs := make(chan struct{}, MAX_CONCURRENT_LS) // Semaphore for limiting concurrent 'ls' calls
	fileChan := make(chan string, 100)              // Buffered channel to collect file paths found

	var wgCollector sync.WaitGroup
	wgCollector.Add(1)
	// Start a goroutine to collect file paths from the channel
	go func() {
		defer wgCollector.Done()
		for file := range fileChan {
			size++
			if filesToCopy.Path == "" {
				filesToCopy.Path = file
			} else {
				next := &FilePath{Path: file, Head: filesToCopy.Head}
				filesToCopy.Next = next
				filesToCopy = *next
			}
		}
	}()

	for _, baseDir := range deviceBaseDirs {
		// Clean the baseDir path to standardize for visitedDirs map and path joining
		cleanBaseDir := filepath.Clean(baseDir)
		if cleanBaseDir == "." {
			cleanBaseDir = "/"
		}
		cleanBaseDir = strings.TrimSuffix(cleanBaseDir, "/") + "/"

		wgTraversal.Add(1)
		semLs <- struct{}{} // Acquire a slot in the 'ls' semaphore
		go func(dir string) {
			defer wgTraversal.Done()
			defer func() { <-semLs }()

			traverseAndFilter(adbPath, dir, fileChan, semLs, &wgTraversal)
		}(cleanBaseDir)
	}

	wgTraversal.Wait()
	close(fileChan)

	wgCollector.Wait()

	return *filesToCopy.Head, size, nil
}
