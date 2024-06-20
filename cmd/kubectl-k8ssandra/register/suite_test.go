package register

import (
	"os"

	"github.com/k8ssandra/k8ssandra-client/internal/envtest"
)

var (
	multiEnv *envtest.MultiK8sEnvironment
	testDir  string
	err      error
)

func startKind() (deferFunc func()) {
	testDir, err = os.MkdirTemp("", "k8ssandra-client-test")
	if err != nil {
		panic(err.Error())
	}
	return envtest.RunMultiKind(func(e *envtest.MultiK8sEnvironment) {
		multiEnv = e
	}, []int{1, 1}, testDir)
}
