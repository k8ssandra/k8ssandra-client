package util

import (
	"os"
	"path/filepath"
)

const (
	dirSuffix = "k8ssandra"
)

// GetCacheDir returns the caching directory for module
func GetCacheDir(module string) (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	targetDir := filepath.Join(userCacheDir, dirSuffix, module)
	return targetDir, nil
}

// GetConfigDir returns the config directory for k8ssandra and creates it if it does not exists
func GetConfigDir(module string) (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	targetDir := filepath.Join(userConfigDir, dirSuffix, module)
	return CreateIfNotExistsDir(targetDir)
}

func CreateIfNotExistsDir(targetDir string) (string, error) {
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return "", err
		}
	}
	return targetDir, nil
}

func VerifyFileExists(yamlPath string) (bool, error) {
	if _, err := os.Stat(yamlPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
