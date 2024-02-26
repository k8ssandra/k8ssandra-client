package cassdcutil

import (
	"context"
	"time"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	"github.com/k8ssandra/k8ssandra-client/pkg/tasks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	waitutil "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CassManager struct {
	client client.Client
}

func NewManager(client client.Client) *CassManager {
	return &CassManager{
		client: client,
	}
}

// CassandraDatacenter fetches the CassandraDatacenter by its name and namespace
func (c *CassManager) CassandraDatacenter(ctx context.Context, name, namespace string) (*cassdcapi.CassandraDatacenter, error) {
	cassdcKey := types.NamespacedName{Namespace: namespace, Name: name}
	cassdc := &cassdcapi.CassandraDatacenter{}

	if err := c.client.Get(ctx, cassdcKey, cassdc); err != nil {
		return nil, err
	}

	return cassdc, nil
}

// CassandraDatacenterPods returns the pods of the CassandraDatacenter
func (c *CassManager) CassandraDatacenterPods(ctx context.Context, cassdc *cassdcapi.CassandraDatacenter) (*corev1.PodList, error) {
	// What if same namespace has two datacenters with the same name? Can that happen?
	podList := &corev1.PodList{}
	err := c.client.List(ctx, podList, client.InNamespace(cassdc.Namespace), client.MatchingLabels(map[string]string{cassdcapi.DatacenterLabel: cassdc.Name}))
	return podList, err
}

// ModifyStoppedState either stops or starts the cluster and does nothing if the state is already as requested
func (c *CassManager) ModifyStoppedState(ctx context.Context, name, namespace string, stop, wait bool) error {
	cassdc, err := c.CassandraDatacenter(ctx, name, namespace)
	if err != nil {
		return err
	}

	cassdc = cassdc.DeepCopy()

	cassdc.Spec.Stopped = stop
	if err := c.client.Update(ctx, cassdc); err != nil {
		// r.Log.Error(err, "failed to update the cassandradatacenter", "CassandraDatacenter", cassdcKey)
		// return ctrl.Result{RequeueAfter: 10 * time.Second}, err
		return err
	}

	if wait {
		if stop {
			if err := waitutil.PollUntilContextTimeout(ctx, 10*time.Second, 10*time.Minute, true, func(context.Context) (bool, error) {
				return c.RefreshStatus(ctx, cassdc, cassdcapi.DatacenterStopped, corev1.ConditionTrue)
			}); err != nil {
				return err
			}

			// And wait for it to finish..
			return waitutil.PollUntilContextTimeout(ctx, 10*time.Second, 10*time.Minute, true, func(context.Context) (bool, error) {
				return c.RefreshStatus(ctx, cassdc, cassdcapi.DatacenterReady, corev1.ConditionFalse)
			})
		}

		if err := waitutil.PollUntilContextTimeout(ctx, 10*time.Second, 10*time.Minute, true, func(context.Context) (bool, error) {
			return c.RefreshStatus(ctx, cassdc, cassdcapi.DatacenterStopped, corev1.ConditionFalse)
		}); err != nil {
			return err
		}

		// And wait for it to finish..
		return waitutil.PollUntilContextTimeout(ctx, 10*time.Second, 10*time.Minute, true, func(context.Context) (bool, error) {
			return c.RefreshStatus(ctx, cassdc, cassdcapi.DatacenterReady, corev1.ConditionTrue)
		})
	}

	return nil
}

func (c *CassManager) RefreshStatus(ctx context.Context, cassdc *cassdcapi.CassandraDatacenter, status cassdcapi.DatacenterConditionType, wanted corev1.ConditionStatus) (bool, error) {
	cassdc, err := c.CassandraDatacenter(ctx, cassdc.Name, cassdc.Namespace)
	if err != nil {
		return false, err
	}

	return cassdc.Status.GetConditionStatus(status) == wanted, nil
}

func (c *CassManager) RestartDc(ctx context.Context, name, namespace, rack string, wait bool) error {
	cassdc, err := c.CassandraDatacenter(ctx, name, namespace)
	if err != nil {
		return err
	}

	task, err := tasks.CreateRestartTask(ctx, c.client, cassdc, rack)
	if err != nil {
		return err
	}

	if wait {
		err = tasks.WaitForCompletion(ctx, c.client, task)
		if err != nil {
			return err
		}
	}
	return nil
}
