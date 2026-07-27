package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/k8ssandra/k8ssandra-client/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	configBuilderExample = `
	# Process the config files from cass-operator input
	%[1]s build [<args>]

	# Process the config files with overrides for directories
	%[1]s build --input <input-dir> --output <output-dir> [<args>]

	# Process sidecar.yaml
	%[1]s build --sidecar --input <input-dir> --output <output-dir>
	`
)

type builderOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams

	inputDir  string
	outputDir string
	sidecar   bool

	configBuilderOptions []config.BuilderOption
}

func newBuilderOptions(streams genericclioptions.IOStreams) *builderOptions {
	return &builderOptions{
		configFlags:          genericclioptions.NewConfigFlags(true),
		IOStreams:            streams,
		configBuilderOptions: make([]config.BuilderOption, 0, 1),
	}
}

// NewCmd provides a cobra command wrapping newAddOptions
func NewBuilderCmd(streams genericclioptions.IOStreams) *cobra.Command {
	return newBuilderCmd(newBuilderOptions(streams))
}

func newBuilderCmd(o *builderOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "build [flags]",
		Short:        "Build config files from cass-operator input",
		Example:      fmt.Sprintf(configBuilderExample, "kubectl k8ssandra config"),
		SilenceUsage: true,
		PreRunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&o.inputDir, "input", "", "read config files from this directory instead of default")
	fl.StringVar(&o.outputDir, "output", "", "write config files to this directory instead of default")
	fl.BoolVar(&o.sidecar, "sidecar", false, "process sidecar configuration files")
	o.configFlags.AddFlags(fl)
	return cmd
}

// Complete parses the arguments and necessary flags to options
func (c *builderOptions) Complete(cmd *cobra.Command, args []string) error {
	// TODO Instead of pkg doing the Getenv parameters, we should probably do them here
	//	    since it's related to the command line interface and shell. Makes it easier to
	// 		refactor later to more sane input
	if c.sidecar {
		c.configBuilderOptions = append(c.configBuilderOptions, config.WithSidecar())
	}
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (c *builderOptions) Validate() error {
	if !c.sidecar {
		return nil
	}

	if c.inputDir == "" || c.outputDir == "" {
		return errors.New("--input and --output are required when using --sidecar")
	}

	return nil
}

// Run processes the input, creates a connection to Kubernetes and processes a secret to add the users
func (c *builderOptions) Run() error {
	ctx := context.Background()

	builder := config.NewBuilder(c.inputDir, c.outputDir, c.configBuilderOptions...)
	return builder.Build(ctx)
}
