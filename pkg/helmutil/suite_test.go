package helmutil_test

import (
	"os"
	"testing"

	"github.com/k8ssandra/k8ssandra-client/internal/envtest"
)

var (
	env *envtest.Environment
)

func TestMain(m *testing.M) {
	os.Exit(envtest.Run(m, func(e *envtest.Environment) { env = e }))
}
