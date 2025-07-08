package settings

type Config struct {
	DirsToCopy []string `json:"dirs_to_copy"`
	DirsToSkip []string `json:"dirs_to_skip"`
	TargetDir  string   `json:"target_dir"`
	SourceDir  string   `json:"source_dir"`
}

const VERSION_FILE_NAME = "version"
const CONFIG_FILE_NAME = "config.json"

const RELEASES_URL = "https://api.github.com/repos/the-main-thing/phone-media-copier/releases/latest"
const WINDOWS_BINARY_NAME = "phone-media-copier-windows-amd64.exe"
const LINUX_BINARY_NAME = "phone-media-copier-linux-amd64"


