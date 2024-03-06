package tasks

import (
	"context"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	controlapi "github.com/k8ssandra/cass-operator/apis/control/v1alpha1"
	k8ssandrataskapi "github.com/k8ssandra/k8ssandra-operator/apis/control/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Restart

func CreateRestartTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string) (*controlapi.CassandraTask, error) {
	args := restartArguments(rackName)
	return CreateTask(ctx, kubeClient, controlapi.CommandRestart, dc, args)
}

func restartArguments(rackName string) *controlapi.JobArguments {
	args := &controlapi.JobArguments{
		RackName: rackName,
	}

	return args
}

func CreateClusterRestartTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, rackName string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args := restartArguments(rackName)
	return CreateClusterTask(ctx, kubeClient, controlapi.CommandRestart, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// Replace

func CreateReplaceTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, podName string) (*controlapi.CassandraTask, error) {
	args, err := replaceArguments(podName)
	if err != nil {
		return nil, err
	}

	return CreateTask(ctx, kubeClient, controlapi.CommandReplaceNode, dc, args)
}

func replaceArguments(podName string) (*controlapi.JobArguments, error) {
	if podName == "" {
		return nil, fmt.Errorf("podName must be specified")
	}
	return &controlapi.JobArguments{PodName: podName}, nil
}

func CreateClusterReplaceTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, podName string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args, err := replaceArguments(podName)
	if err != nil {
		return nil, err
	}

	return CreateClusterTask(ctx, kubeClient, controlapi.CommandReplaceNode, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// Flush

func CreateFlushTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string, podName string) (*controlapi.CassandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateTask(ctx, kubeClient, controlapi.CommandFlush, dc, args)
}

func CreateClusterFlushTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, rackName, podName string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateClusterTask(ctx, kubeClient, controlapi.CommandFlush, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// Cleanup

func CreateCleanupTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string, podName string) (*controlapi.CassandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateTask(ctx, kubeClient, controlapi.CommandCleanup, dc, args)
}

func CreateClusterCleanupTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, rackName, podName string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateClusterTask(ctx, kubeClient, controlapi.CommandCleanup, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// UpgradeSSTables

func CreateUpgradeSSTablesTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string, podName string) (*controlapi.CassandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateTask(ctx, kubeClient, controlapi.CommandUpgradeSSTables, dc, args)
}

func CreateClusterUpgradeSSTablesTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, rackName, podName string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateClusterTask(ctx, kubeClient, controlapi.CommandUpgradeSSTables, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// Scrub

func CreateScrubTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string, podName string) (*controlapi.CassandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateTask(ctx, kubeClient, controlapi.CommandScrub, dc, args)
}

func CreateClusterScrubTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, rackName, podName string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateClusterTask(ctx, kubeClient, controlapi.CommandScrub, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// Compaction

func CreateCompactionTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string, podName, keyspaceName string, tables []string) (*controlapi.CassandraTask, error) {
	args, err := compactionArguments(rackName, podName, keyspaceName, tables)
	if err != nil {
		return nil, err
	}
	return CreateTask(ctx, kubeClient, controlapi.CommandCompaction, dc, args)
}

func compactionArguments(rackName, podName, keyspaceName string, tables []string) (*controlapi.JobArguments, error) {
	args := commonArguments(rackName, podName)
	args.KeyspaceName = keyspaceName

	if keyspaceName == "" && len(tables) > 0 {
		return nil, fmt.Errorf("keyspace must be specified when tables are specified")
	}

	if len(tables) > 0 {
		args.Tables = tables
	}

	return args, nil
}

func CreateClusterCompactionTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, rackName, podName, keyspaceName string, tables []string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args, err := compactionArguments(rackName, podName, keyspaceName, tables)
	if err != nil {
		return nil, err
	}
	return CreateClusterTask(ctx, kubeClient, controlapi.CommandCompaction, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// Move

// GarbageCollect

func CreateGCTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string, podName string) (*controlapi.CassandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateTask(ctx, kubeClient, controlapi.CommandGarbageCollect, dc, args)
}

func CreateClusterGCTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, rackName, podName string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args := commonArguments(rackName, podName)
	return CreateClusterTask(ctx, kubeClient, controlapi.CommandGarbageCollect, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// Rebuild

func CreateRebuildTask(ctx context.Context, kubeClient client.Client, dc *cassdcapi.CassandraDatacenter, rackName string, podName, sourceDatacenter string) (*controlapi.CassandraTask, error) {
	args, err := rebuildArguments(rackName, podName, sourceDatacenter)
	if err != nil {
		return nil, err
	}
	return CreateTask(ctx, kubeClient, controlapi.CommandRebuild, dc, args)
}

func rebuildArguments(rackName, podName, sourceDatacenter string) (*controlapi.JobArguments, error) {
	args := commonArguments(rackName, podName)
	if sourceDatacenter == "" {
		return nil, fmt.Errorf("sourceDatacenter must be specified")
	}
	args.SourceDatacenter = sourceDatacenter

	return args, nil
}

func CreateClusterRebuildTask(ctx context.Context, kubeClient client.Client, namespace, cluster, dcName, rackName, podName, sourceDatacenter string, dcConcurrencyPolicy, taskConcurrencyPolicy batchv1.ConcurrencyPolicy) (*k8ssandrataskapi.K8ssandraTask, error) {
	args, err := rebuildArguments(rackName, podName, sourceDatacenter)
	if err != nil {
		return nil, err
	}
	return CreateClusterTask(ctx, kubeClient, controlapi.CommandRebuild, namespace, cluster, []string{dcName}, args, dcConcurrencyPolicy, taskConcurrencyPolicy)
}

// Assistance methods

func commonArguments(rackName, podName string) *controlapi.JobArguments {
	return &controlapi.JobArguments{
		RackName: rackName,
		PodName:  podName,
	}
}

func CreateTask(ctx context.Context, kubeClient client.Client, command controlapi.CassandraCommand, dc *cassdcapi.CassandraDatacenter, args *controlapi.JobArguments) (*controlapi.CassandraTask, error) {
	task := &controlapi.CassandraTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      createName(dc.Name, string(command)),
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
