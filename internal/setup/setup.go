package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"phone-copier/internal/settings"
	"phone-copier/internal/version"
)

func Setup(configDir string) (string, error) {
	versonFilePath := filepath.Join(configDir, settings.VERSION_FILE_NAME)
	configFilePath := filepath.Join(configDir, settings.CONFIG_FILE_NAME)
	_, err := os.Stat(configDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil {
		currentVersion, err := os.ReadFile(versonFilePath)
		if err != nil {
			return "", err
		}
		return string(currentVersion), nil
	}

	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(versonFilePath, []byte(version.VERSION), 0644)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(settings.Config{}, "", "    ")
	if err != nil {
		return "", err
	}
	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		return "", err
	}
	return version.VERSION, nil
}
