package envtest

import (
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"
)

type MultiK8sEnvironment []*Environment

func RunMulti(m *testing.M, setupFunc func(e *MultiK8sEnvironment), numClusters int) (code int) {
	e := make(MultiK8sEnvironment, numClusters)
	ctx := ctrl.SetupSignalHandler()
	for i := 0; i < numClusters; i++ {
		e[i] = NewEnvironment(ctx)
		e[i].Start()
	}
	defer func() {
		for i := 0; i < numClusters; i++ {
			e[i].Stop()
		}
	}()
	setupFunc(&e)
	exitCode := m.Run()
	return exitCode
}
