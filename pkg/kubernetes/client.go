package kubernetes

import (
	"context"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespacedClient encapsulates namespacedClient with public namespace and restConfig
type NamespacedClient struct {
	client.Client
	Config    *rest.Config
	Namespace string
}

// GetClient returns a controller-runtime client with cass-operator API defined
func GetClient(restConfig *rest.Config) (client.Client, error) {
	c, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, err
	}

	err = cassdcapi.AddToScheme(c.Scheme())

	return c, err
}

func GetClientInNamespace(restConfig *rest.Config, namespace string) (NamespacedClient, error) {
	c, err := GetClient(restConfig)
	if err != nil {
		return NamespacedClient{}, err
	}

	c = client.NewNamespacedClient(c, namespace)
	return NamespacedClient{
		Config: restConfig,
		Client: c,
	}, nil
	// return c, nil
}

func CreateNamespaceIfNotExists(ctx context.Context, client client.Client, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	if err := client.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
