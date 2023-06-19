package config

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClientOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

// NewClientOptions provides an instance of ClientOptions with default values
func NewClientOptions(streams genericclioptions.IOStreams) *ClientOptions {
	return &ClientOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping ClientOptions
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewClientOptions(streams)

	cmd := &cobra.Command{
		Use: "config [subcommand] [flags]",
	}

	// Add subcommands
	cmd.AddCommand(NewBuilderCmd(streams))
	// TODO Add the idea of allowing to modify cassandra-yaml with interactive editor from the
	// command line

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}
