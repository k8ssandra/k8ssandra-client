package cassdcutil

import (
	"context"
	"time"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	"github.com/k8ssandra/k8ssandra-client/pkg/tasks"
	corev1 "k8s.io/api/core/v1"
	waitutil "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultPollInterval = 1 * time.Second
	defaultTimeout      = 10 * time.Minute
)

type CassManager struct {
	client client.Client
}

func NewManager(client client.Client) *CassManager {
	return &CassManager{
		client: client,
	}
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
			if err := c.WaitForStatus(ctx, cassdc, cassdcapi.DatacenterStopped, corev1.ConditionTrue, defaultPollInterval, defaultTimeout); err != nil {
				return err
			}

			if err := c.WaitForStatus(ctx, cassdc, cassdcapi.DatacenterReady, corev1.ConditionFalse, defaultPollInterval, defaultTimeout); err != nil {
				return err
			}
		} else {
			if err := c.WaitForStatus(ctx, cassdc, cassdcapi.DatacenterStopped, corev1.ConditionFalse, defaultPollInterval, defaultTimeout); err != nil {
				return err
			}

			if err := c.WaitForStatus(ctx, cassdc, cassdcapi.DatacenterReady, corev1.ConditionTrue, defaultPollInterval, defaultTimeout); err != nil {
				return err
			}
		}
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

// RestartDc creates a task to restart the cluster and waits for completion if wait is set to true
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

func (c *CassManager) WaitForStatus(ctx context.Context, cassdc *cassdcapi.CassandraDatacenter, status cassdcapi.DatacenterConditionType, wanted corev1.ConditionStatus, interval, timeout time.Duration) error {
	if interval == 0 {
		interval = defaultPollInterval
	}

	if timeout == 0 {
		timeout = defaultTimeout
	}

	if err := waitutil.PollUntilContextTimeout(ctx, interval, timeout, true, func(context.Context) (bool, error) {
		return c.RefreshStatus(ctx, cassdc, status, wanted)
	}); err != nil {
		return err
	}
	return nil
}
