package scheduler

import (
	"context"
	"fmt"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/backend/cache"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	plfeature "k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/interpodaffinity"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeaffinity"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/tainttoleration"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubehelpers "github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
)

type ClientBuilder interface {
	GetClient() (client.Client, error)
	GetClientset() (kubernetes.Interface, error)
}

func NewKubernetesClientBuilder(config *rest.Config) ClientBuilder {
	return &KubernetesClientBuilder{
		config: config,
	}
}

type KubernetesClientBuilder struct {
	config *rest.Config
}

func (c *KubernetesClientBuilder) GetClient() (client.Client, error) {
	return kubehelpers.GetClient(c.config)
}

func (c *KubernetesClientBuilder) GetClientset() (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(c.config)
}

func createNodeInfos(ctx context.Context, cli client.Client, nodes []*corev1.Node) ([]*framework.NodeInfo, error) {
	nodeInfos := make([]*framework.NodeInfo, 0, len(nodes))

	for _, node := range nodes {
		nodeInfo := framework.NewNodeInfo()
		nodeInfo.SetNode(node)
		pods := &corev1.PodList{}
		if err := cli.List(ctx, pods, client.MatchingFields{"spec.nodeName": nodeInfo.Node().Name}); err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {
			nodeInfo.AddPod(&pod)
		}

		nodeInfos = append(nodeInfos, nodeInfo)
	}

	return nodeInfos, nil
}

func fetchNodes(ctx context.Context, cli client.Client) ([]*corev1.Node, error) {
	nodeList := &corev1.NodeList{}
	if err := cli.List(ctx, nodeList); err != nil {
		return nil, err
	}

	nodes := make([]*corev1.Node, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		nodes = append(nodes, &node)
	}

	return nodes, nil
}

func fetchPods(ctx context.Context, cli client.Client) ([]*corev1.Pod, error) {
	podList := &corev1.PodList{}
	if err := cli.List(ctx, podList); err != nil {
		return nil, err
	}

	pods := make([]*corev1.Pod, 0, len(podList.Items))
	for _, pod := range podList.Items {
		pods = append(pods, &pod)
	}

	return pods, nil
}

// TryScheduling checks if the proposed pods will be schedulable in the cluster. The ProposedPods must have their
// labels, tolerations, affinities and resource limits set
func TryScheduling(ctx context.Context, builder ClientBuilder, proposedPods []*corev1.Pod) error {
	cli, err := builder.GetClient()
	if err != nil {
		return err
	}

	clientset, err := builder.GetClientset()
	if err != nil {
		return err
	}

	nodes, err := fetchNodes(ctx, cli)
	if err != nil {
		return err
	}

	pods, err := fetchPods(ctx, cli)
	if err != nil {
		return err
	}

	state := framework.NewCycleState()

	schedulablePlugin, err := nodeunschedulable.New(ctx, nil, nil, plfeature.Features{})
	if err != nil {
		return err
	}

	noderesourcesPlugin, err := noderesources.NewFit(ctx, &config.NodeResourcesFitArgs{ScoringStrategy: defaultScoringStrategy}, nil, plfeature.Features{})
	if err != nil {
		return err
	}

	nodeaffinityPlugin, err := nodeaffinity.New(ctx, &config.NodeAffinityArgs{}, nil, plfeature.Features{})
	if err != nil {
		return err
	}

	snapshot := cache.NewSnapshot(pods, nodes)

	usableNodes, err := snapshot.NodeInfos().List()
	if err != nil {
		return err
	}

	informerFactory := informers.NewSharedInformerFactory(clientset, 0)

	fh, err := frameworkruntime.NewFramework(ctx, nil, nil, frameworkruntime.WithSnapshotSharedLister(snapshot), frameworkruntime.WithInformerFactory(informerFactory))
	if err != nil {
		return err
	}

	informerFactory.Start(ctx.Done())
	informerFactory.WaitForCacheSync(ctx.Done())

	interpodaffinityPlugin, err := interpodaffinity.New(ctx, &config.InterPodAffinityArgs{}, fh, plfeature.Features{})
	if err != nil {
		return err
	}

	tainttolerationPlugin, err := tainttoleration.New(ctx, nil, nil, plfeature.Features{})
	if err != nil {
		return err
	}

	plugins := []framework.FilterPlugin{
		noderesourcesPlugin.(framework.FilterPlugin),
		schedulablePlugin.(framework.FilterPlugin),
		nodeaffinityPlugin.(framework.FilterPlugin),
		interpodaffinityPlugin.(framework.FilterPlugin),
		tainttolerationPlugin.(framework.FilterPlugin),
	}

	succeededPods := 0
NextPod:
	for _, pod := range proposedPods {
	NextNode:
		for _, node := range usableNodes {
			podInfo, err := framework.NewPodInfo(pod)
			if err != nil {
				return err
			}

			for _, plugin := range plugins {
				var preFilterStatus, filterStatus *framework.Status
				if prefilterPlugin, ok := plugin.(framework.PreFilterPlugin); ok {
					_, preFilterStatus = prefilterPlugin.PreFilter(ctx, state, podInfo.Pod)
					// if preFilterStatus.IsSkip() {
					// 	fmt.Printf("prefilter %s skipped pod %s/%s: not schedulable on node %s, reason: %s\n", plugin.Name(), pod.Name, pod.Namespace, node.Node().Name, preFilterStatus.Message())
					// }
					if preFilterStatus.Code() != framework.Success && preFilterStatus.Code() != framework.Skip {
						fmt.Printf("prefilter %s rejected pod %s/%s: not schedulable on node %s, reason: %s, state: %v\n", plugin.Name(), pod.Name, pod.Namespace, node.Node().Name, preFilterStatus.Message(), preFilterStatus.Code())
						continue NextNode
					}
				}
				if preFilterStatus != nil && preFilterStatus.Code() == framework.Skip {
					continue
				}
				filterStatus = plugin.Filter(ctx, state, podInfo.Pod, node)
				if filterStatus.Code() != framework.Success {
					fmt.Printf("filter %s rejected pod %s/%s: not schedulable on node %s, reason: %s\n", plugin.Name(), pod.Name, pod.Namespace, node.Node().Name, filterStatus.Message())
					continue NextNode
				}
			}

			// TODO Since in 1.29 we can't use the interpodnodeaffinity, we will instead limit
			// the amount of pods that can be scheduled to a node to 1. As such, next round will not
			// be able to schedule anything to this node.
			node.AddPod(pod)
			n := node.Node()
			n.Spec.Unschedulable = true
			succeededPods++

			continue NextPod
		}
		// Pod was never added to any node
		return fmt.Errorf("unable to schedule all the pods, requested: %d, schedulable: %d", len(proposedPods), succeededPods)
	}
	return nil
}

var defaultScoringStrategy = &config.ScoringStrategy{
	Type: config.LeastAllocated,
	Resources: []config.ResourceSpec{
		{Name: "cpu", Weight: 1},
		{Name: "memory", Weight: 1},
	},
}
