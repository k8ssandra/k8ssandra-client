package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetAllKubernetesNodeIPAddresses returns a mapping of IPAddress -> node_name
func GetAllKubernetesNodeIPAddresses(ctx context.Context, cli client.Client) (map[string]string, error) {
	nodes := &corev1.NodeList{}
	if err := cli.List(ctx, nodes); err != nil {
		return nil, err
	}

	ipAddresses := make(map[string]string, len(nodes.Items))

	for _, node := range nodes.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				ipAddresses[addr.Address] = node.Name
			}
		}
	}

	return ipAddresses, nil
}
