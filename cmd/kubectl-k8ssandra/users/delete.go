package users

import (
	"context"
	"fmt"

	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	"github.com/k8ssandra/k8ssandra-client/pkg/users"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	userDeleteExample = `
	# Delete users from CassandraDatacenter
	%[1]s delete [<args>]

	# Delete user tryme from CassandraDatacenter dc1
	%[1]s delete --dc dc1 --username tryme
	`
)

type deleteOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	namespace  string
	datacenter string

	// For manual entering from CLI
	username string
}

func newDeleteOptions(streams genericclioptions.IOStreams) *deleteOptions {
	return &deleteOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping newAddOptions
func NewDeleteCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newDeleteOptions(streams)

	cmd := &cobra.Command{
		Use:     "delete [flags]",
		Short:   "Delete user from CassandraDatacenter installation",
		Example: fmt.Sprintf(userDeleteExample, "kubectl k8ssandra users"),
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
	fl.StringVar(&o.datacenter, "dc", "", "target datacenter")
	fl.StringVarP(&o.username, "username", "u", "", "username to add")
	o.configFlags.AddFlags(fl)
	return cmd
}

// Complete parses the arguments and necessary flags to options
func (c *deleteOptions) Complete(cmd *cobra.Command, args []string) error {
	var err error

	c.namespace, _, err = c.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	if c.username == "" && len(args) > 0 {
		c.username = args[0]
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (c *deleteOptions) Validate() error {
	if c.datacenter == "" {
		return errNoDcDc
	}

	if c.username == "" {
		return errMissingUsername
	}

	return nil
}

// Run processes the input, creates a connection to Kubernetes and processes a secret to add the users
func (c *deleteOptions) Run() error {
	restConfig, err := c.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.GetClientInNamespace(restConfig, c.namespace)
	if err != nil {
		return err
	}

	ctx := context.Background()

	return users.Delete(ctx, kubeClient, c.datacenter, c.username)
}
