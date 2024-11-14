package mgmtapi

import (
	"context"
	"net"
	"net/http"

	"github.com/k8ssandra/cass-operator/pkg/httphelper"
	"github.com/k8ssandra/k8ssandra-client/pkg/cassdcutil"
	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO We need to detect if we're running inside Kubernetes or not and wrap the client if necessary

// NewManagementClient returns a new instance for management-api go-client
func NewManagementClient(ctx context.Context, client client.Client, namespace, datacenter string) (httphelper.NodeMgmtClient, error) {
	manager := cassdcutil.NewManager(client)
	dc, err := manager.CassandraDatacenter(ctx, datacenter, namespace)
	if err != nil {
		return httphelper.NodeMgmtClient{}, err
	}

	return httphelper.NewMgmtClient(ctx, client, dc, nil)
}

func NewForwardedManagementClient(ctx context.Context, restConfig *rest.Config, namespace, datacenter string) (httphelper.NodeMgmtClient, *corev1.Pod, error) {
	client, err := kubernetes.GetClientInNamespace(restConfig, namespace)
	if err != nil {
		return httphelper.NodeMgmtClient{}, nil, err
	}

	manager := cassdcutil.NewManager(client)

	dc, err := manager.CassandraDatacenter(ctx, datacenter, namespace)
	if err != nil {
		return httphelper.NodeMgmtClient{}, nil, err
	}

	pod, err := manager.FirstRunningDatacenterPod(ctx, dc)
	if err != nil {
		return httphelper.NodeMgmtClient{}, pod, err
	}

	forwardAddr, err := kubernetes.PortForwardMgmtPort(restConfig, pod)
	if err != nil {
		return httphelper.NodeMgmtClient{}, pod, err
	}

	customTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial(network, forwardAddr)
		},
	}

	mgmtClient, err := httphelper.NewMgmtClient(ctx, client, dc, customTransport)
	return mgmtClient, pod, err
}
