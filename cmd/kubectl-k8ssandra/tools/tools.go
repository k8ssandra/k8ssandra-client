package tools

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClientOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

// NewToolsOptions provides an instance of NamespaceOptions with default values
func NewToolsOptions(streams genericclioptions.IOStreams) *ClientOptions {
	return &ClientOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewToolsCmd provides a cobra command wrapping ClientOptions
func NewToolsCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewToolsOptions(streams)

	cmd := &cobra.Command{
		Use: "tools [subcommand] [flags]",
	}

	// Add subcommands
	cmd.AddCommand(NewEstimateCmd(streams))

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}
