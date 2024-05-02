package envtest

import (
	"os"
	"path/filepath"
	"runtime"
)

// https://stackoverflow.com/questions/31873396/is-it-possible-to-get-the-current-root-of-package-structure-as-a-string-in-golan
func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(b), "../..")
}

func GetBuildDir() string {
	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		_, b, _, _ := runtime.Caller(0)
		buildDir = filepath.Join(filepath.Dir(b), "../..")
	}
	return buildDir
}
