package secrets

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// readTargetPath supports two different formats for users. If the target is a file, it must be in the format
// username=password, if it's a directory, then it must follow the Kubernetes secret format,
// filename = username, file = password
func ReadTargetPath(path string) (map[string]string, error) {
	f, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if f.IsDir() {
		return readTargetSecretMount(path)
	}
	return readTargetFile(path)
}

// readTargetSecretMount is processing the old standard set by cass-operator
// this method can only parse a single username/password pair
func readTargetSecretMount(path string) (map[string]string, error) {
	var username, password string

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Couldn't access the file for some reason
			return err
		}

		if d.IsDir() {
			// This will be walked later
			return nil
		}

		switch d.Name() {
		case "password":
			password, err = readFile(path)
			return err
		case "username":
			username, err = readFile(path)
			return err
		}

		return nil
	})

	return map[string]string{username: password}, err
}

func readTargetFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	users := make(map[string]string)

	// Remove the comment lines to reduce the ConfigMap size
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		userInfo := strings.SplitN(line, "=", 2)
		if len(userInfo) > 1 {
			users[userInfo[0]] = userInfo[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// Creates a directory on the local filesystem at the path:
// {path}/{secret}
func CreateSecretsDirectory(path string, secret string) error {
	secretPath := fmt.Sprintf("%v/%v", path, secret)
	err := os.MkdirAll(secretPath, 0750)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

// Writes key-values for a kubernetes secret to path {path}/{secret} whee each key
// is its own file whose contents is the value of the key i.e. filename = key, file = value
func WriteSecretsKeyValue(path string, secret string, key string, value string) error {
	f, err := os.Create(fmt.Sprintf("%s/%s/%s", path, secret, key))
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString(value)
	return err
}

func readFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer f.Close()

	fileContents, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(fileContents), nil
}
