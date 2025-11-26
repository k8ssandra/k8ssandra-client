// mostly inspired by https://github.com/milesbxf/empathy/blob/master/cluster.go

package envtest

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kindapi "sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"
)

var kindConfig = `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
`

type KindManager struct {
	ClusterName        string
	Nodes              int
	KubeconfigLocation *os.File
	client             *client.Client
	RestConfig         *rest.Config
}

func (k *KindManager) CreateKindCluster() error {
	log.Println("Creating kind cluster", k.ClusterName)
	provider := cluster.NewProvider(cluster.ProviderWithLogger(kindcmd.NewLogger()))
	kindConfig, err := setWorkerNodes(k.Nodes)

	if err != nil {
		return err
	}

	err = provider.Create(
		k.ClusterName,
		cluster.CreateWithNodeImage(""),
		cluster.CreateWithRetain(false),
		cluster.CreateWithWaitForReady(time.Duration(0)),
		cluster.CreateWithKubeconfigPath(k.KubeconfigLocation.Name()),
		cluster.CreateWithDisplayUsage(false),
		cluster.CreateWithRawConfig(kindConfig),
	)
	if err != nil {
		return err
	}
	config, err := clientcmd.BuildConfigFromFlags("", k.KubeconfigLocation.Name())
	if err != nil {
		return err
	}
	kClient, err := client.New(config, client.Options{})
	if err != nil {
		return err
	}
	k.client = &kClient
	k.RestConfig = config
	return err
}

func (k *KindManager) TearDownKindCluster() error {
	provider := cluster.NewProvider(cluster.ProviderWithLogger(kindcmd.NewLogger()))
	return provider.Delete(k.ClusterName, k.KubeconfigLocation.Name())
}

func setWorkerNodes(workers int) ([]byte, error) {
	cluster := &kindapi.Cluster{}
	err := yaml.Unmarshal([]byte(kindConfig), cluster)
	if err != nil {
		return nil, err
	}
	cluster.Nodes = []kindapi.Node{
		{Role: kindapi.ControlPlaneRole},
	}
	for i := 0; i < workers; i++ {
		cluster.Nodes = append(cluster.Nodes, kindapi.Node{Role: kindapi.WorkerRole})
	}
	return yaml.Marshal(cluster)
}

func (k *KindManager) GetClient() (*client.Client, error) {
	if k.client != nil {
		return k.client, nil
	}

	config, err := clientcmd.BuildConfigFromFlags("", k.KubeconfigLocation.Name())
	if err != nil {
		return nil, err
	}

	kclient, err := client.New(config, client.Options{})
	k.client = &kclient
	if err != nil {
		return nil, err
	}

	return k.client, nil
}
