package envtest

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	controlapi "github.com/k8ssandra/cass-operator/apis/control/v1alpha1"
	k8ssandrataskapi "github.com/k8ssandra/k8ssandra-operator/apis/control/v1alpha1"

	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Run(m *testing.M, setupFunc func(e *Environment)) (code int) {
	env := NewEnvironment()
	env.start()
	setupFunc(env)
	exitCode := m.Run()
	env.stop()
	return exitCode
}

type Environment struct {
	intClient     client.Client
	env           *envtest.Environment
	cancelManager context.CancelFunc
	Context       context.Context
}

func NewEnvironment() *Environment {
	env := &Environment{}
	env.env = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join(RootDir(), "testfiles", "crd")},
		ErrorIfCRDPathMissing: true,
	}

	ctx := ctrl.SetupSignalHandler()
	ctx, cancel := context.WithCancel(ctx)
	env.Context = ctx
	env.cancelManager = cancel
	return env
}

// https://stackoverflow.com/questions/31873396/is-it-possible-to-get-the-current-root-of-package-structure-as-a-string-in-golan
func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(b), "../..")
}

func (e *Environment) Client(namespace string) client.Client {
	return client.NewNamespacedClient(e.intClient, namespace)
}

func (e *Environment) start() {
	cfg, err := e.env.Start()
	if err != nil {
		panic(err)
	}

	if err := cassdcapi.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}

	if err := controlapi.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}

	if err := k8ssandrataskapi.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}

	//+kubebuilder:scaffold:scheme

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(err)
	}

	e.intClient = k8sClient
}

func (e *Environment) stop() {
	e.cancelManager()
	if err := e.env.Stop(); err != nil {
		panic(err)
	}
}

func (e *Environment) CreateNamespace(t *testing.T) string {
	namespace := strings.ToLower(t.Name())
	if err := kubernetes.CreateNamespaceIfNotExists(e.intClient, namespace); err != nil {
		t.FailNow()
	}

	return namespace
}

func (e *Environment) RestConfig() *rest.Config {
	return e.env.Config
}
