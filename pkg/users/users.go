package users

import (
	"context"

	"github.com/k8ssandra/k8ssandra-client/pkg/cassdcutil"
	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	"github.com/k8ssandra/k8ssandra-client/pkg/mgmtapi"
	"github.com/k8ssandra/k8ssandra-client/pkg/secrets"

	corev1 "k8s.io/api/core/v1"
)

func AddNewUsersFromSecret(ctx context.Context, c kubernetes.NamespacedClient, datacenter string, secretPath string, superusers bool) error {
	// Create ManagementClient
	mgmtClient, err := mgmtapi.NewManagementClient(ctx, c, c.Namespace, datacenter)
	if err != nil {
		return err
	}

	pod, err := targetPod(ctx, c, datacenter)
	if err != nil {
		return err
	}

	users, err := secrets.ReadTargetPath(secretPath)
	if err != nil {
		return err
	}

	for user, pass := range users {
		if err := mgmtClient.CallCreateRoleEndpoint(pod, user, pass, superusers); err != nil {
			return err
		}
	}

	return nil
}

func targetPod(ctx context.Context, c kubernetes.NamespacedClient, datacenter string) (*corev1.Pod, error) {
	cassManager := cassdcutil.NewManager(c)
	dc, err := cassManager.CassandraDatacenter(ctx, datacenter, c.Namespace)
	if err != nil {
		return nil, err
	}

	podList, err := cassManager.CassandraDatacenterPods(ctx, dc)
	if err != nil {
		return nil, err
	}

	return &podList.Items[0], nil
}

func AddNewUser(ctx context.Context, c kubernetes.NamespacedClient, datacenter string, username string, password string, superuser bool) error {
	mgmtClient, err := mgmtapi.NewManagementClient(ctx, c, c.Namespace, datacenter)
	if err != nil {
		return err
	}

	pod, err := targetPod(ctx, c, datacenter)
	if err != nil {
		return err
	}

	if err := mgmtClient.CallCreateRoleEndpoint(pod, username, password, superuser); err != nil {
		return err
	}

	return nil
}
