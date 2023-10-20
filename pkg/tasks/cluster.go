package tasks

import (
	"context"
	"fmt"
	"time"

	controlapi "github.com/k8ssandra/cass-operator/apis/control/v1alpha1"
	k8ssandrataskapi "github.com/k8ssandra/k8ssandra-operator/apis/control/v1alpha1"
	k8ssandraapi "github.com/k8ssandra/k8ssandra-operator/apis/k8ssandra/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateClusterTask(ctx context.Context, kubeClient client.Client, command controlapi.CassandraCommand, kc *k8ssandraapi.K8ssandraCluster, args *controlapi.JobArguments) (*k8ssandrataskapi.K8ssandraTask, error) {
	generatedName := fmt.Sprintf("%s-%s-%d", kc.Name, command, time.Now().Unix())
	task := &k8ssandrataskapi.K8ssandraTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generatedName,
			Namespace: kc.Namespace,
		},
		Spec: k8ssandrataskapi.K8ssandraTaskSpec{
			Cluster: corev1.ObjectReference{
				Name:      kc.Name,
				Namespace: kc.Namespace,
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

	if args != nil {
		task.Spec.Template.Jobs[0].Arguments = *args
	}

	if err := kubeClient.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}
