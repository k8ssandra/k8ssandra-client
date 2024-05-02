package envtest

import (
	"context"
	"os"
	"path/filepath"

	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeployImageConfig(cli client.Client) error {
	configPath := filepath.Join(RootDir(), "testfiles", "image_config.yaml")
	b, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if err := kubernetes.CreateNamespaceIfNotExists(cli, "mission-control"); err != nil {
		return err
	}

	confMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cass-operator-manager-config",
			Namespace: "mission-control",
		},
		Data: map[string]string{
			"image_config.yaml": string(b),
		},
	}

	if err := cli.Create(context.TODO(), confMap); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
