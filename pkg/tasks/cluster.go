package tasks

import (
	"context"
	"fmt"

	controlapi "github.com/k8ssandra/cass-operator/apis/control/v1alpha1"
	k8ssandrataskapi "github.com/k8ssandra/k8ssandra-operator/apis/control/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateClusterTask(ctx context.Context, kubeClient client.Client, command controlapi.CassandraCommand, namespace, kcName string, datacenters []string, args *controlapi.JobArguments) (*k8ssandrataskapi.K8ssandraTask, error) {
	if kcName == "" || namespace == "" {
		return nil, fmt.Errorf("clusterName and namespace must be specified")
	}

	task := &k8ssandrataskapi.K8ssandraTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      createName(kcName, string(command)),
			Namespace: namespace,
		},
		Spec: k8ssandrataskapi.K8ssandraTaskSpec{
			Cluster: corev1.ObjectReference{
				Name:      kcName,
				Namespace: namespace,
			},
			Template: controlapi.CassandraTaskTemplate{
				Jobs: []controlapi.CassandraJob{
					{
						Command: command,
					},
				},
			},
		},
	}

	if len(datacenters) > 0 {
		task.Spec.Datacenters = datacenters
	}

	if args != nil {
		task.Spec.Template.Jobs[0].Arguments = *args
	}

	if err := kubeClient.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}
