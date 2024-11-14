package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/k8ssandra/k8ssandra-client/pkg/users"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	userListExample = `
	# List users of CassandraDatacenter or K8ssandraCluster
	%[1]s list [<args>]

	# List users of CassandraDatacenter dc1
	%[1]s list --dc dc1

	# List users of K8ssandraCluster cluster1
	%[1]s list --cluster cluster1
	`

	errNoDcOrCluster = errors.New("either cluster or datacenter target is required")
)

type listOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	namespace  string
	datacenter string
	cluster    string
}

func newListOptions(streams genericclioptions.IOStreams) *listOptions {
	return &listOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping newAddOptions
func NewListCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newListOptions(streams)

	cmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   "List users of CassandraDatacenter or K8ssandraCluster installation",
		Example: fmt.Sprintf(userListExample, "kubectl k8ssandra users"),
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
	// fl.StringVar(&o.cluster, "cluster", "", "target cluster")
	o.configFlags.AddFlags(fl)
	return cmd
}

// Complete parses the arguments and necessary flags to options
func (c *listOptions) Complete(cmd *cobra.Command, args []string) error {
	var err error

	c.namespace, _, err = c.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (c *listOptions) Validate() error {
	if c.datacenter == "" && c.cluster == "" {
		return errNoDcOrCluster
	}

	return nil
}

// Run processes the input, creates a connection to Kubernetes and processes a secret to add the users
func (c *listOptions) Run() error {
	restConfig, err := c.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	ctx := context.Background()

	users, err := users.List(ctx, restConfig, c.namespace, c.datacenter)
	if err != nil {
		return err
	}

	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Superuser", Width: 10},
		{Title: "Login", Width: 7},
		{Title: "Options", Width: 10},
		{Title: "Datacenters", Width: 10},
	}

	rows := make([]table.Row, 0, len(users))

	for _, user := range users {
		row := table.Row{
			user.Name,
			user.Super,
			user.Login,
			user.Options,
			user.Datacenters,
		}
		rows = append(rows, row)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	fmt.Print(t.View())

	return nil
}
