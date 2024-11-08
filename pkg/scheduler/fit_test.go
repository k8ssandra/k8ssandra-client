package scheduler

import (
	"context"
	"testing"

	api "github.com/k8ssandra/cass-operator/apis/cassandra/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSmokeResources(t *testing.T) {
	require := require.New(t)
	ctx := context.TODO()
	cli := createClient()
	require.NoError(cli.Create(ctx, makeNode("node1")))

	pods := []*corev1.Pod{
		makePod("pod1", makeResources(100, 100, 1)),
	}
	require.NoError(TryScheduling(ctx, cli, pods))

	// Lets add more resources than the node can handle
	pods = []*corev1.Pod{
		makePod("pod1", makeResources(1100, 1100, 1)),
	}

	require.Error(TryScheduling(ctx, cli, pods))
}

func TestSmokeTolerations(t *testing.T) {
	require := require.New(t)
	ctx := context.TODO()
	cli := createClient()
	n := makeNode("node1")
	n.Spec.Taints = []corev1.Taint{
		{
			Key:    "cassandra.datastax.com/node-purpose",
			Value:  "database",
			Effect: corev1.TaintEffectNoSchedule,
		},
	}

	require.NoError(cli.Create(ctx, n))

	pods := []*corev1.Pod{
		makePod("pod1", makeResources(100, 100, 1)),
	}

	// While we have enough resources, we do not have any tolerations set and as such should fail
	// the scheduling
	require.Error(TryScheduling(ctx, cli, pods))

	pods[0].Spec.Tolerations = []corev1.Toleration{
		{
			Key:   "cassandra.datastax.com/node-purpose",
			Value: "database",
		},
	}

	require.NoError(TryScheduling(ctx, cli, pods))
}

func TestSmokeNodeAffinity(t *testing.T) {
	require := require.New(t)
	ctx := context.TODO()
	cli := createClient()
	n := makeNode("node1")
	require.NoError(cli.Create(ctx, n))

	pod := makePod("pod1", makeResources(100, 100, 1))

	pod.Spec.Affinity = &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "cassandra.datastax.com/node-purpose",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{"database"},
							},
						},
					},
				},
			},
		},
	}
	pods := []*corev1.Pod{
		pod,
	}

	require.Error(TryScheduling(ctx, cli, pods))

	metav1.SetMetaDataLabel(&n.ObjectMeta, "cassandra.datastax.com/node-purpose", "database")
	require.NoError(cli.Update(ctx, n))

	require.NoError(TryScheduling(ctx, cli, pods))
}

func TestSmokeInternodePodAffinity(t *testing.T) {
	require := require.New(t)
	ctx := context.TODO()
	cli := createClient()
	require.NoError(cli.Create(ctx, makeNode("node1")))
	pod := makePod("pod1", makeResources(100, 100, 1))

	antiAffinity := corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
			{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      api.ClusterLabel,
							Operator: metav1.LabelSelectorOpExists,
						},
						{
							Key:      api.DatacenterLabel,
							Operator: metav1.LabelSelectorOpExists,
						},
						{
							Key:      api.RackLabel,
							Operator: metav1.LabelSelectorOpExists,
						},
					},
				},
				TopologyKey: "kubernetes.io/hostname",
			},
		},
	}

	pod.Spec.Affinity = &corev1.Affinity{
		PodAntiAffinity: &antiAffinity,
	}

	pod2 := makePod("pod2", makeResources(100, 100, 1))
	pod2.Spec.Affinity = &corev1.Affinity{
		PodAntiAffinity: &antiAffinity,
	}

	pods := []*corev1.Pod{
		pod,
		pod2,
	}

	// We should be able to schedule only a single pod, since the other one would run into
	// an issue with the interpod anti-affinity
	require.Error(TryScheduling(ctx, cli, pods))
}

func createClient() client.Client {
	s := runtime.NewScheme()
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Node{})

	nodeNameIndexer := client.IndexerFunc(func(obj client.Object) []string {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}
		return []string{pod.Spec.NodeName}
	})

	fakeClient := fake.NewClientBuilder().WithIndex(&corev1.Pod{}, "spec.nodeName", nodeNameIndexer).Build()
	return fakeClient
}

func makeResources(milliCPU, memory, pods int64) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(memory, resource.BinarySI),
		corev1.ResourcePods:   *resource.NewQuantity(pods, resource.DecimalSI),
	}
}

func makeNode(name string) *corev1.Node {
	n := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
		},
	}
	n.Status.Capacity, n.Status.Allocatable = makeResources(1000, 1000, 100), makeResources(1000, 1000, 100)
	return n
}

func makePod(name string, resources corev1.ResourceList) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: resources,
					},
				},
			},
		},
	}
}
