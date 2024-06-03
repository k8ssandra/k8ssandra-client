package register

import (
	"os"
	"testing"

	"github.com/k8ssandra/k8ssandra-client/internal/envtest"
)

var (
	multiEnv *envtest.MultiK8sEnvironment
	testDir  string
	err      error
)

func TestMain(m *testing.M) {
	testDir, err = os.MkdirTemp("", "k8ssandra-client-test")
	if err != nil {
		panic(err.Error())
	}
	os.Exit(envtest.RunMultiKind(m, func(e *envtest.MultiK8sEnvironment) {
		multiEnv = e
	}, []int{1, 1}, testDir))
}
