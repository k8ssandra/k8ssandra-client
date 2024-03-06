package tasks_test

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	"testing"

	cassdcapi "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	controlapi "github.com/k8ssandra/cass-operator/apis/control/v1alpha1"
	"github.com/stretchr/testify/assert"

	"github.com/k8ssandra/k8ssandra-client/pkg/tasks"
)

func TestCreateRestartTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	rackName := "rack1"

	task, err := tasks.CreateRestartTask(context.Background(), kubeClient, dc, rackName)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandRestart, task.Spec.Jobs[0].Command)
}

func TestCreateClusterRestartTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	rackName := "rack1"

	task, err := tasks.CreateClusterRestartTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, batchv1.ForbidConcurrent, batchv1.AllowConcurrent)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandRestart, task.Spec.Template.Jobs[0].Command)
	assert.Equal(t, batchv1.ForbidConcurrent, task.Spec.DcConcurrencyPolicy)
	assert.Equal(t, batchv1.AllowConcurrent, task.Spec.Template.ConcurrencyPolicy)
}

func TestCreateReplaceTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	podName := "pod1"

	task, err := tasks.CreateReplaceTask(context.Background(), kubeClient, dc, podName)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandReplaceNode, task.Spec.Jobs[0].Command)

	_, err = tasks.CreateReplaceTask(context.Background(), kubeClient, dc, "")
	assert.Error(t, err)
}

func TestCreateClusterReplaceTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	podName := "pod1"

	task, err := tasks.CreateClusterReplaceTask(context.Background(), kubeClient, namespace, cluster, dcName, podName, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandReplaceNode, task.Spec.Template.Jobs[0].Command)

	_, err = tasks.CreateClusterReplaceTask(context.Background(), kubeClient, namespace, cluster, dcName, "", "", "")
	assert.Error(t, err)
}

func TestCreateFlushTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateFlushTask(context.Background(), kubeClient, dc, rackName, podName)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandFlush, task.Spec.Jobs[0].Command)
}

func TestCreateClusterFlushTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateClusterFlushTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, podName, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandFlush, task.Spec.Template.Jobs[0].Command)
}

func TestCreateCleanupTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateCleanupTask(context.Background(), kubeClient, dc, rackName, podName)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandCleanup, task.Spec.Jobs[0].Command)
}

func TestCreateClusterCleanupTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateClusterCleanupTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, podName, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandCleanup, task.Spec.Template.Jobs[0].Command)
}

func TestCreateUpgradeSSTablesTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateUpgradeSSTablesTask(context.Background(), kubeClient, dc, rackName, podName)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandUpgradeSSTables, task.Spec.Jobs[0].Command)
}

func TestCreateClusterUpgradeSSTablesTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateClusterUpgradeSSTablesTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, podName, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandUpgradeSSTables, task.Spec.Template.Jobs[0].Command)
}

func TestCreateScrubTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateScrubTask(context.Background(), kubeClient, dc, rackName, podName)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandScrub, task.Spec.Jobs[0].Command)
}

func TestCreateClusterScrubTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateClusterScrubTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, podName, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandScrub, task.Spec.Template.Jobs[0].Command)
}

func TestCreateCompactionTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	rackName := "rack1"
	podName := "pod1"
	keyspaceName := "test-keyspace"
	tables := []string{"table1", "table2"}

	task, err := tasks.CreateCompactionTask(context.Background(), kubeClient, dc, rackName, podName, keyspaceName, tables)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandCompaction, task.Spec.Jobs[0].Command)

	// Keyspace should be required if tables is set
	_, err = tasks.CreateCompactionTask(context.Background(), kubeClient, dc, rackName, podName, "", tables)
	assert.Error(t, err)

	// But without both is accepted
	task, err = tasks.CreateCompactionTask(context.Background(), kubeClient, dc, rackName, podName, "", nil)

	assert.NoError(t, err)
	assert.NotNil(t, task)
}

func TestCreateClusterCompactionTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	rackName := "rack1"
	podName := "pod1"
	keyspaceName := "test-keyspace"
	tables := []string{"table1", "table2"}

	task, err := tasks.CreateClusterCompactionTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, podName, keyspaceName, tables, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandCompaction, task.Spec.Template.Jobs[0].Command)

	// Only cluster is really required
	task, err = tasks.CreateClusterCompactionTask(context.Background(), kubeClient, namespace, cluster, "", "", "", "", nil, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
}

func TestCreateGCTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateGCTask(context.Background(), kubeClient, dc, rackName, podName)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandGarbageCollect, task.Spec.Jobs[0].Command)
}

func TestCreateClusterGCTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	rackName := "rack1"
	podName := "pod1"

	task, err := tasks.CreateClusterGCTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, podName, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandGarbageCollect, task.Spec.Template.Jobs[0].Command)
}

func TestCreateRebuildTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	rackName := "rack1"
	podName := "pod1"
	sourceDatacenter := "dc1"

	task, err := tasks.CreateRebuildTask(context.Background(), kubeClient, dc, rackName, podName, sourceDatacenter)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandRebuild, task.Spec.Jobs[0].Command)

	// Empty sourceDatacenter should result in validation error
	_, err = tasks.CreateRebuildTask(context.Background(), kubeClient, dc, rackName, podName, "")
	assert.Error(t, err)
}

func TestCreateClusterRebuildTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := "test-dc"
	rackName := "rack1"
	podName := "pod1"
	sourceDatacenter := "dc1"

	task, err := tasks.CreateClusterRebuildTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, podName, sourceDatacenter, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandRebuild, task.Spec.Template.Jobs[0].Command)

	_, err = tasks.CreateClusterRebuildTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, podName, "", "", "")
	assert.Error(t, err)
}

func TestCreateTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	command := controlapi.CommandRestart
	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "test-dc"
	args := &controlapi.JobArguments{}

	task, err := tasks.CreateTask(context.Background(), kubeClient, command, dc, args)

	assert.NoError(t, err)
	assert.NotNil(t, task)
}

func TestCreateTaskLongName(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	command := controlapi.CommandRestart
	dc := &cassdcapi.CassandraDatacenter{}
	dc.Name = "you-never-know-what-kind-of-names-people-come-up-with-for-no-reason-at-all"
	args := &controlapi.JobArguments{}

	task, err := tasks.CreateTask(context.Background(), kubeClient, command, dc, args)

	assert.NoError(t, err)
	assert.NotNil(t, task)
}

func TestCreateClusterWideTask(t *testing.T) {
	namespace := env.CreateNamespace(t)
	kubeClient := env.Client(namespace)

	cluster := "test-cluster"
	dcName := ""
	rackName := "rack1"

	task, err := tasks.CreateClusterRestartTask(context.Background(), kubeClient, namespace, cluster, dcName, rackName, "", "")

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, controlapi.CommandRestart, task.Spec.Template.Jobs[0].Command)
	assert.Equal(t, 0, len(task.Spec.Datacenters))
}
