package users

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

/*
	users list
	users delete (does mgmt-api have this ability?)
*/

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
		Use: "users [subcommand] [flags]",
	}

	// Add subcommands
	cmd.AddCommand(NewAddCmd(streams))
	cmd.AddCommand(NewDeleteCmd(streams))
	cmd.AddCommand(NewListCmd(streams))
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}
