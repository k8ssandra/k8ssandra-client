package register

import (
	"os"
	"testing"

	"github.com/k8ssandra/k8ssandra-client/internal/envtest"
)

var (
	multiEnv *envtest.MultiK8sEnvironment
)

func TestMain(m *testing.M) {
	os.Exit(envtest.RunMultiKind(m, func(e *envtest.MultiK8sEnvironment) {
		multiEnv = e
	}, []int{1, 1}))
}
