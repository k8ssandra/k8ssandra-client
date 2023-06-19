package config

import (
	"context"
	"fmt"

	"github.com/k8ssandra/k8ssandra-client/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	configBuilderExample = `
	# Process the config files from cass-operator input
	%[1]s build [<args>]
	`
)

type builderOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

func newBuilderOptions(streams genericclioptions.IOStreams) *builderOptions {
	return &builderOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping newAddOptions
func NewBuilderCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newBuilderOptions(streams)

	cmd := &cobra.Command{
		Use:     "build [flags]",
		Short:   "Build config files from cass-operator input",
		Example: fmt.Sprintf(configBuilderExample, "kubectl k8ssandra config"),
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
	o.configFlags.AddFlags(fl)
	return cmd
}

// Complete parses the arguments and necessary flags to options
func (c *builderOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (c *builderOptions) Validate() error {
	return nil
}

// Run processes the input, creates a connection to Kubernetes and processes a secret to add the users
func (c *builderOptions) Run() error {
	ctx := context.Background()

	return config.Build(ctx)
}
