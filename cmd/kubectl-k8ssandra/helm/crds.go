package helm

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/k8ssandra/k8ssandra-client/pkg/helmutil"
	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	upgraderExample = `
	# update CRDs in the namespace to chartVersion
	%[1]s upgrade --chartName <chartName> --chartVersion <chartVersion> [<args>]

	# update CRDs in the namespace to chartVersion with non-default chartRepo (helm.k8ssandra.io)
	%[1]s upgrade --chartName <chartName> --chartVersion <chartVersion> --chartRepo <repository> [<args>]
	`
	errNotEnoughParameters = fmt.Errorf("not enough parameters, requires chartName and chartVersion")
)

type options struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	namespace    string
	chartName    string
	chartVersion string
	chartRepo    string
	repoURL      string
	download     bool
}

func newOptions(streams genericclioptions.IOStreams) *options {
	return &options{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping cqlShOptions
func NewUpgradeCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(streams)

	cmd := &cobra.Command{
		Use:          "upgrade [flags]",
		Short:        "upgrade CRDs from chart to target version",
		Example:      fmt.Sprintf(upgraderExample, "kubectl k8ssandra helm crds"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				log.Error("Error upgrading CustomResourceDefinitions", "error", err)
				return err
			}

			return nil
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&o.chartName, "chartName", "", "chartName to upgrade")
	fl.StringVar(&o.chartVersion, "chartVersion", "", "chartVersion to upgrade to")
	fl.StringVar(&o.chartRepo, "chartRepo", "", "optional chart repository name to override the default (k8ssandra)")
	fl.StringVar(&o.repoURL, "repoURL", "", "optional chart repository url to override the default (helm.k8ssandra.io)")
	fl.BoolVar(&o.download, "download", false, "only download the chart")
	o.configFlags.AddFlags(fl)

	return cmd
}

// Complete parses the arguments and necessary flags to options
func (c *options) Complete(cmd *cobra.Command, args []string) error {
	var err error
	if c.chartName == "" && c.chartVersion == "" {
		return errNotEnoughParameters
	}

	if c.repoURL == "" {
		c.repoURL = helmutil.StableK8ssandraRepoURL
	}

	if c.chartRepo == "" {
		c.chartRepo = helmutil.K8ssandraRepoName
	}

	c.namespace, _, err = c.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

// Validate ensures that all required arguments and flag values are provided
func (c *options) Validate() error {
	// TODO Validate that the chartVersion is valid
	return nil
}

// Run removes the finalizers for a release X in the given namespace
func (c *options) Run() error {
	restConfig, err := c.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	var kubeClient kubernetes.NamespacedClient
	if !c.download {
		kubeClient, err = kubernetes.GetClientInNamespace(restConfig, c.namespace)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()

	upgrader, err := helmutil.NewUpgrader(kubeClient, c.chartRepo, c.repoURL, c.chartName)
	if err != nil {
		return err
	}

	_, err = upgrader.Upgrade(ctx, c.chartVersion)
	return err
}
