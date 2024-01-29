package helm

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClientOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

// NewClientOptions provides an instance of NamespaceOptions with default values
func NewHelmOptions(streams genericclioptions.IOStreams) *ClientOptions {
	return &ClientOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping NamespaceOptions
func NewHelmCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewHelmOptions(streams)

	cmd := &cobra.Command{
		Use: "helm [subcommand] [flags]",
	}

	// Add subcommands
	cmd.AddCommand(NewUpgradeCmd(streams))

	// cmd.Flags().BoolVar(&o.listNamespaces, "list", o.listNamespaces, "if true, print the list of all namespaces in the current KUBECONFIG")
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}
