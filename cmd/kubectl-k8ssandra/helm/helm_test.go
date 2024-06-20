package helm

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestSimplestCRDCommand(t *testing.T) {
	require := require.New(t)

	cmd := NewUpgradeCmd(genericiooptions.NewTestIOStreamsDiscard())
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	cmd.Root().SetArgs([]string{"upgrade", "--chartName", "k8ssandra-operator", "--chartVersion", "1.0.0"})
	require.NoError(cmd.Execute())
}

func TestMissingParamsCRDCommand(t *testing.T) {
	require := require.New(t)

	cmd := NewUpgradeCmd(genericiooptions.NewTestIOStreamsDiscard())
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	cmd.Root().SetArgs([]string{"upgrade", "--chartName", "k8ssandra-operator"})
	require.Error(cmd.Execute())
}

func TestInvalidParamsCRDCommand(t *testing.T) {
	require := require.New(t)

	cmd := NewUpgradeCmd(genericiooptions.NewTestIOStreamsDiscard())
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	cmd.Root().SetArgs([]string{"upgrade", "--chartName", "k8ssandra-operator", "--chartTarget", "1.0.0"})
	require.Error(cmd.Execute())
}

func TestAllParamsCRDCommand(t *testing.T) {
	require := require.New(t)

	cmd := NewUpgradeCmd(genericiooptions.NewTestIOStreamsDiscard())
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	cmd.Root().SetArgs([]string{"upgrade", "--chartName", "k8ssandra-operator", "--chartVersion", "1.0.0",
		"--chartRepo", "devel", "--repoURL", "https://helm.k8ssandra.io/devel", "--download", "true", "--charts", "cass-operator", "--charts", "k8ssandra-operator,cass-operator"})
	require.NoError(cmd.Execute())
}
