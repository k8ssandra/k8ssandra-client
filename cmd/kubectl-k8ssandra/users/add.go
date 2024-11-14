package users

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k8ssandra/k8ssandra-client/pkg/kubernetes"
	"github.com/k8ssandra/k8ssandra-client/pkg/ui"
	"github.com/k8ssandra/k8ssandra-client/pkg/users"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	userAddExample = `
	# Add new users to CassandraDatacenter
	%[1]s add [<args>]

	# Add new user example to CassandraDatacenter dc2 with password prompting
	%[1]s add --dc dc1 --username example --superuser

	# Add new superusers to CassandraDatacenter dc1 from a path /tmp/users.txt
	%[1]s add --dc dc1 --path /tmp/users.txt --superuser
	`
	errNoDcDc           = fmt.Errorf("target CassandraDatacenter is required")
	errDoubleDefinition = fmt.Errorf("either --path or --username is allowed, not both")
	errMissingUsername  = fmt.Errorf("--username is required")
)

type addOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
	namespace  string
	datacenter string
	superuser  bool

	// For manual entering from CLI
	username string
	password string

	// When reading from files
	secretPath string
}

func newAddOptions(streams genericclioptions.IOStreams) *addOptions {
	return &addOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// NewCmd provides a cobra command wrapping newAddOptions
func NewAddCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newAddOptions(streams)

	cmd := &cobra.Command{
		Use:     "add [flags]",
		Short:   "Add new users to CassandraDatacenter installation",
		Example: fmt.Sprintf(userAddExample, "kubectl k8ssandra users"),
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
	fl.StringVar(&o.secretPath, "path", "", "path to users data")
	fl.StringVar(&o.datacenter, "dc", "", "target datacenter")
	fl.BoolVar(&o.superuser, "superuser", true, "create users as superusers") // TODO Set default to false
	fl.StringVarP(&o.username, "username", "u", "", "username to add")
	fl.StringVarP(&o.password, "password", "p", "", "password to set for the user")
	o.configFlags.AddFlags(fl)
	return cmd
}

// Complete parses the arguments and necessary flags to options
func (c *addOptions) Complete(cmd *cobra.Command, args []string) error {
	var err error

	c.namespace, _, err = c.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (c *addOptions) Validate() error {
	if c.datacenter == "" {
		return errNoDcDc
	}

	if c.secretPath != "" && c.username != "" {
		return errDoubleDefinition
	}

	if c.password != "" && c.username == "" {
		return errMissingUsername
	}

	return nil
}

// Run processes the input, creates a connection to Kubernetes and processes a secret to add the users
func (c *addOptions) Run() error {
	restConfig, err := c.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.GetClientInNamespace(restConfig, c.namespace)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if c.secretPath != "" {
		return users.AddNewUsersFromSecret(ctx, kubeClient, c.datacenter, c.secretPath, c.superuser)
	}

	// Interactive prompt

	prompts := make([]*ui.Prompt, 0, 1)

	userPrompt := ui.NewPrompt("Username")
	passPrompt := ui.NewPrompt("Password").Mask()

	if c.username == "" {
		prompts = append(prompts, userPrompt)
	}

	if c.password == "" {
		prompts = append(prompts, passPrompt)
	}

	if len(prompts) > 0 {
		prompter := ui.NewPrompter(prompts)
		if _, err := tea.NewProgram(prompter).Run(); err != nil {
			return err
		}

		// Parse values
		c.password = passPrompt.Value()
		if c.username == "" {
			c.username = userPrompt.Value()
		}
	}

	return users.Add(ctx, kubeClient, c.datacenter, c.username, c.password, c.superuser)
}
