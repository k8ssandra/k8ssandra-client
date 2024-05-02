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
	// metrics.DefaultBindAddress = "0" This no longer appears to exist...
	os.Exit(envtest.RunMulti(m, func(e *envtest.MultiK8sEnvironment) {
		multiEnv = e
	}, 2))
}
