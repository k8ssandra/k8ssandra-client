package k8ssandra

import (
	// "github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/cleaner"
	// "github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/cqlsh"
	// "github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/crds"
	// "github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/edit"
	// "github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/list"
	// "github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/migrate"
	"github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/config"
	"github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/helm"
	"github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/nodetool"
	"github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/operate"
	"github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/register"
	"github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/tools"
	"github.com/k8ssandra/k8ssandra-client/cmd/kubectl-k8ssandra/users"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClientOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

// NewClientOptions provides an instance of NamespaceOptions with default values
func NewClientOptions(streams genericclioptions.IOStreams) *ClientOptions {
	return &ClientOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping NamespaceOptions
func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewClientOptions(streams)

	cmd := &cobra.Command{
		Use: "k8ssandra [subcommand] [flags]",
	}

	// Add subcommands
	// cmd.AddCommand(cqlsh.NewCmd(streams))
	// cmd.AddCommand(cleaner.NewCmd(streams))
	// cmd.AddCommand(edit.NewCmd(streams))
	cmd.AddCommand(operate.NewStartCmd(streams))
	cmd.AddCommand(operate.NewRestartCmd(streams))
	cmd.AddCommand(operate.NewStopCmd(streams))
	// cmd.AddCommand(list.NewCmd(streams))
	cmd.AddCommand(users.NewCmd(streams))
	cmd.AddCommand(config.NewCmd(streams))
	cmd.AddCommand(helm.NewHelmCmd(streams))
	cmd.AddCommand(nodetool.NewCmd(streams))
	cmd.AddCommand(tools.NewToolsCmd(streams))
	register.SetupRegisterClusterCmd(cmd, streams)

	// cmd.Flags().BoolVar(&o.listNamespaces, "list", o.listNamespaces, "if true, print the list of all namespaces in the current KUBECONFIG")
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}
