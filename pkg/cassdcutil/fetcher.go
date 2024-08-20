package cassdcutil

import (
	"context"
	"fmt"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CassandraDatacenter fetches the CassandraDatacenter by its name and namespace
func (c *CassManager) CassandraDatacenter(ctx context.Context, name, namespace string) (*cassdcapi.CassandraDatacenter, error) {
	cassdcKey := types.NamespacedName{Namespace: namespace, Name: name}
	cassdc := &cassdcapi.CassandraDatacenter{}

	if err := c.client.Get(ctx, cassdcKey, cassdc); err != nil {
		return nil, err
	}

	return cassdc, nil
}

// PodDatacenter returns the CassandraDatacenter instance of the pod if it's managed by cass-operator
func (c *CassManager) PodDatacenter(ctx context.Context, podName, namespace string) (*cassdcapi.CassandraDatacenter, error) {
	key := types.NamespacedName{Namespace: namespace, Name: podName}
	pod := &corev1.Pod{}
	err := c.client.Get(ctx, key, pod)
	if err != nil {
		return nil, err
	}

	if dc, found := pod.Labels[cassdcapi.DatacenterLabel]; !found {
		return nil, fmt.Errorf("target pod not managed by cass-operator, no datacenter label")
	} else {
		// Get CassandraDatacenter for the dc
		cassDcKey := types.NamespacedName{Namespace: namespace, Name: dc}
		cassdc := &cassdcapi.CassandraDatacenter{}
		err = c.client.Get(ctx, cassDcKey, cassdc)
		if err != nil {
			return nil, err
		}

		return cassdc, nil
	}
}

// CassandraDatacenterPods returns the pods of the CassandraDatacenter
func (c *CassManager) CassandraDatacenterPods(ctx context.Context, cassdc *cassdcapi.CassandraDatacenter) (*corev1.PodList, error) {
	// What if same namespace has two datacenters with the same name? Can that happen?
	podList := &corev1.PodList{}
	err := c.client.List(ctx, podList, client.InNamespace(cassdc.Namespace), client.MatchingLabels(map[string]string{cassdcapi.DatacenterLabel: cassdc.Name}))
	return podList, err
}
