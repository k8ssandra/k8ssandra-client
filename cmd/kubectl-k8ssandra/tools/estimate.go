package tools

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	"github.com/k8ssandra/k8ssandra-client/pkg/scheduler"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	estimateExample = `
	# Estimate if pods will fit the cluster
	%[1]s estimate [<args>]

	# Estimate if 4 pods each with 2Gi of memory and 2 vCPUs will be able to run on the cluster.
	# All CPU values and memory use Kubernetes notation
	%[1]s estimate --count 4 --memory 2Gi --cpu 2000m
	`
	errInvalidCount = errors.New("Count of pods must be higher than 0")
)

type estimateOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams

	count  int
	memory string
	cpu    string

	cpuQuantity    resource.Quantity
	memoryQuantity resource.Quantity
}

func newEstimateOptions(streams genericclioptions.IOStreams) *estimateOptions {
	return &estimateOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping newAddOptions
func NewEstimateCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newEstimateOptions(streams)

	cmd := &cobra.Command{
		Use:     "estimate [flags]",
		Short:   "Estimate if datacenter can be expanded",
		Example: fmt.Sprintf(estimateExample, "kubectl k8ssandra tools estimate"),
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	fl := cmd.Flags()
	fl.IntVar(&o.count, "count", 0, "new nodes to create")
	fl.StringVar(&o.cpu, "cpu", "0", "how many cores per node")
	fl.StringVar(&o.memory, "memory", "0", "how much memory per node")

	if err := cmd.MarkFlagRequired("memory"); err != nil {
		panic(err)
	}

	if err := cmd.MarkFlagRequired("cpu"); err != nil {
		panic(err)
	}

	if err := cmd.MarkFlagRequired("count"); err != nil {
		panic(err)
	}

	o.configFlags.AddFlags(fl)
	return cmd
}

// Complete parses the arguments and necessary flags to options
func (c *estimateOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (c *estimateOptions) Validate() error {
	if c.count < 0 {
		return errInvalidCount
	}

	var err error
	cpuQuantity, err := resource.ParseQuantity(c.cpu)
	if err != nil {
		return err
	}

	memoryQuantity, err := resource.ParseQuantity(c.memory)
	if err != nil {
		return err
	}

	c.memoryQuantity = memoryQuantity
	c.cpuQuantity = cpuQuantity

	return nil
}

// Run processes the input, creates a connection to Kubernetes and processes a secret to add the users
func (c *estimateOptions) Run() error {
	restConfig, err := c.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.GetClient(restConfig)
	if err != nil {
		return err
	}
	ctx := context.Background()

	proposedPods := makePods(c.count, makeResources(c.cpuQuantity.MilliValue(), c.memoryQuantity.Value()))

	if err := scheduler.TryScheduling(ctx, kubeClient, proposedPods); err != nil {
		return errors.Wrap(err, "Unable to schedule the pods")
	}

	return nil
}

func makePods(count int, resources corev1.ResourceList) []*corev1.Pod {
	pods := make([]*corev1.Pod, count)
	for i := 0; i < count; i++ {
		pods[i] = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "a",
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

	return pods
}

func makeResources(milliCPU, memory int64) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(memory, resource.BinarySI),
	}
}
