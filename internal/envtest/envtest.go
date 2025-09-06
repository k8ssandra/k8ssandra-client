package envtest

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	controlapi "github.com/k8ssandra/cass-operator/apis/control/v1alpha1"
	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	k8ssandrataskapi "github.com/k8ssandra/k8ssandra-operator/apis/control/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/utils/ptr"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func Run(m *testing.M, setupFunc func(e *Environment)) (code int) {
	ctx := ctrl.SetupSignalHandler()
	env := NewEnvironment(ctx)
	env.Start()
	setupFunc(env)
	exitCode := m.Run()
	env.Stop()
	return exitCode
}

type Environment struct {
	client        client.Client
	env           *envtest.Environment
	cancelManager context.CancelFunc
	Context       context.Context
	Kubeconfig    string
	KindCluster   KindManager
}

func NewEnvironment(ctx context.Context) *Environment {
	env := &Environment{}
	env.env = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(RootDir(), "testfiles", "crd"),
		},
		ErrorIfCRDPathMissing: true,
	}
	ctx, cancel := context.WithCancel(ctx)
	env.Context = ctx
	env.cancelManager = cancel
	return env
}

func NewKindEnvironment(ctx context.Context, cluster KindManager) *Environment {
	err := cluster.CreateKindCluster()
	if err != nil {
		panic(err)
	}
	env := &Environment{}
	env.env = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(RootDir(), "testfiles", "crd"),
		},
		UseExistingCluster:    ptr.To(true),
		ErrorIfCRDPathMissing: true,
		Config:                cluster.RestConfig,
	}
	ctx, cancel := context.WithCancel(ctx)
	env.Context = ctx
	env.cancelManager = cancel
	env.Kubeconfig = cluster.KubeconfigLocation.Name()
	env.env.Config = cluster.RestConfig
	env.KindCluster = cluster
	return env
}

func (e *Environment) GetClientInNamespace(namespace string) client.Client {
	c, err := kubernetes.GetClientInNamespace(e.env.Config, namespace)
	if err != nil {
		panic(err)
	}
	return c
}

func (e *Environment) RestConfig() *rest.Config {
	return e.env.Config
}

func (e *Environment) RawClient() client.Client {
	return e.client
}

func (e *Environment) Start() {
	cfg, err := e.env.Start()
	if err != nil {
		panic(err)
	}

	k8sClient, err := kubernetes.GetClient(cfg)
	if err != nil {
		panic(err)
	}

	if err := cassdcapi.AddToScheme(k8sClient.Scheme()); err != nil {
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

	if err := apiextensions.AddToScheme(scheme.Scheme); err != nil {
		panic(err)
	}

	e.client = k8sClient
}

func (e *Environment) Stop() {
	e.cancelManager()
	if err := e.env.Stop(); err != nil {
		panic(err)
	}
}

func (e *Environment) CreateNamespace(t *testing.T) string {
	namespace := strings.ToLower(t.Name())
	if err := kubernetes.CreateNamespaceIfNotExists(t.Context(), e.client, namespace); err != nil {
		t.FailNow()
	}

	return namespace
}

func (e *Environment) GetKubeconfig() ([]byte, error) {
	clientConfig, err := CreateKubeconfigFileForRestConfig(e.env.Config)
	if err != nil {
		return nil, err
	}
	return clientcmd.Write(clientConfig)
}

func CreateKubeconfigFileForRestConfig(restConfig *rest.Config) (clientcmdapi.Config, error) {
	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default-cluster"] = &clientcmdapi.Cluster{
		Server:                   restConfig.Host,
		CertificateAuthorityData: restConfig.CAData,
	}
	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default-context"] = &clientcmdapi.Context{
		Cluster:  "default-cluster",
		AuthInfo: "default-user",
	}
	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos["default-user"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: restConfig.CertData,
		ClientKeyData:         restConfig.KeyData,
	}
	clientConfig := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: "default-context",
		AuthInfos:      authinfos,
	}

	return clientConfig, nil
}
