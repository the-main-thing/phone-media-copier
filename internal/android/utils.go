package android

import (
	"path/filepath"
	"strings"
)

func getFileType(ext string) FileType {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".webp", ".heic", ".bmp", ".tiff":
		return FileTypeImage
	case ".mp4", ".mov", ".avi", ".webm", ".mkv", ".3gp", ".wmv":
		return FileTypeVideo
	case ".mp3", ".wav", ".ogg", ".aac", ".flac", ".m4a", ".opus":
		return FileTypeAudio
	default:
		return FileTypeUnknown
	}
}

func isMediaExtension(ext string) bool {
	return getFileType(ext) != FileTypeUnknown
}

func sanitizeFileName(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

func getFilePath(sourcePath string, targetDir string, suffix string) string {
	lowerCasePath := strings.ToLower(sourcePath)
	filename := sanitizeFileName(filepath.Base(sourcePath))
	ext := filepath.Ext(filename)
	fileType := getFileType(ext)
	var dirname string
	switch fileType {
	case FileTypeImage:
		dirname = "Pictures"
		break
	case FileTypeVideo:
		dirname = "Videos"
		break
	case FileTypeAudio:
		dirname = "Music"
		break
	default:
		dirname = "Unhandled"
		break
	}
	if suffix != "" {
		nameWithoutExt := strings.TrimSuffix(filename, ext)
		filename = nameWithoutExt + suffix + ext
	}
	if strings.Contains(lowerCasePath, "whatsapp") {
		return filepath.Join(targetDir, "WhatsApp", dirname, filename)
	}
	if strings.Contains(lowerCasePath, "telegram") {
		return filepath.Join(targetDir, "Telegram", dirname, filename)
	}

	return filepath.Join(targetDir, dirname, filename)
}
