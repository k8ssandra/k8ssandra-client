package scheduler

import (
	"context"
	"fmt"

	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	plfeature "k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeaffinity"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/tainttoleration"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createNodeInfos(ctx context.Context, cli client.Client, nodes []corev1.Node) ([]*framework.NodeInfo, error) {
	nodeInfos := make([]*framework.NodeInfo, 0, len(nodes))

	for _, node := range nodes {
		nodeInfo := framework.NewNodeInfo()
		nodeInfo.SetNode(&node)
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

func fetchNodes(ctx context.Context, cli client.Client) ([]corev1.Node, error) {
	nodes := corev1.NodeList{}
	if err := cli.List(ctx, &nodes); err != nil {
		return nil, err
	}

	return nodes.Items, nil
}

// TryScheduling checks if the proposed pods will be schedulable in the cluster. The ProposedPods must have their
// labels, tolerations, affinities and resource limits set
func TryScheduling(ctx context.Context, cli client.Client, proposedPods []*corev1.Pod) error {
	nodes, err := fetchNodes(ctx, cli)
	if err != nil {
		return err
	}

	usableNodes, err := createNodeInfos(ctx, cli, nodes)
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

	// fh, err := frameworkruntime.NewFramework(ctx, nil, nil,
	// 	frameworkruntime.WithSnapshotSharedLister(sharedLister),
	// 	frameworkruntime.WithInformerFactory(informerFactory))

	// interpodaffinityPlugin, err := interpodaffinity.New(ctx, &config.InterPodAffinityArgs{}, fh)
	// if err != nil {
	// 	return err
	// }

	tainttolerationPlugin, err := tainttoleration.New(ctx, nil, nil, plfeature.Features{})
	if err != nil {
		return err
	}

	plugins := []framework.FilterPlugin{
		noderesourcesPlugin.(framework.FilterPlugin),
		schedulablePlugin.(framework.FilterPlugin),
		nodeaffinityPlugin.(framework.FilterPlugin),
		// interpodaffinityPlugin.(framework.FilterPlugin),
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
					if preFilterStatus.Code() != framework.Success && preFilterStatus.Code() != framework.Skip {
						continue NextNode
					}
				}
				filterStatus = plugin.Filter(ctx, state, podInfo.Pod, node)
				if filterStatus.Code() != framework.Success {
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
