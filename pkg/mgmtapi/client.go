package mgmtapi

import (
	"context"

	"github.com/k8ssandra/cass-operator/pkg/httphelper"
	"github.com/k8ssandra/k8ssandra-client/pkg/cassdcutil"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewManagementClient returns a new instance for management-api go-client
func NewManagementClient(ctx context.Context, client client.Client, namespace, datacenter string) (httphelper.NodeMgmtClient, error) {
	manager := cassdcutil.NewManager(client)
	dc, err := manager.CassandraDatacenter(ctx, datacenter, namespace)
	if err != nil {
		return httphelper.NodeMgmtClient{}, err
	}

	return httphelper.NewMgmtClient(ctx, client, dc, nil)
}
