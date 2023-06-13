package tasks

import (
	"context"
	"fmt"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	controlapi "github.com/k8ssandra/cass-operator/apis/control/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateRestartTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string) (*controlapi.CassandraTask, error) {
	args := controlapi.JobArguments{}
	if rackName != "" {
		args.RackName = rackName
	}

	return CreateTask(ctx, kubeClient, controlapi.CommandRestart, dc, &args)
}

func CreateTask(ctx context.Context, kubeClient client.Client, command controlapi.CassandraCommand, dc *cassdcapi.CassandraDatacenter, args *controlapi.JobArguments) (*controlapi.CassandraTask, error) {
	generatedName := fmt.Sprintf("%s-command-time?", dc.Name)
	task := &controlapi.CassandraTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generatedName,
			Namespace: dc.Namespace,
		},
		Spec: controlapi.CassandraTaskSpec{
			Datacenter: corev1.ObjectReference{
				Name:      dc.Name,
				Namespace: dc.Namespace,
			},
			CassandraTaskTemplate: controlapi.CassandraTaskTemplate{
				Jobs: []controlapi.CassandraJob{
					{
						Name:    fmt.Sprintf("%s-%s", dc.Name, string(command)),
						Command: command,
					},
				},
			},
		},
	}
	if args != nil {
		task.Spec.Jobs[0].Arguments = *args
	}

	if err := kubeClient.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}
