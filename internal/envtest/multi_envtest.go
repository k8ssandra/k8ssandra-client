package envtest

import (
	"os"
	"strconv"
	"sync"
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

func RunMultiKind(m *testing.M, setupFunc func(e *MultiK8sEnvironment), topology []int) (code int) {
	e := make(MultiK8sEnvironment, len(topology))
	ctx := ctrl.SetupSignalHandler()
	var wg sync.WaitGroup
	for i := 0; i < len(topology); i++ {
		cluster := KindManager{
			ClusterName:        "cluster" + strconv.Itoa(i),
			KubeconfigLocation: os.NewFile(0, GetBuildDir()+"/cluster"+strconv.Itoa(i)),
			Nodes:              topology[i],
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			e[i] = NewKindEnvironment(ctx, cluster)
			e[i].Start()
		}(i)
	}
	wg.Wait()
	defer func() {
		for i := 0; i < len(topology); i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				if err := e[i].KindCluster.TearDownKindCluster(); err != nil {
					panic(err)
				}
				e[i].Stop()
			}(i)
		}
		wg.Wait()
	}()

	setupFunc(&e)
	exitCode := m.Run()
	return exitCode
}
