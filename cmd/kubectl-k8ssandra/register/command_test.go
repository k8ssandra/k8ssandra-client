package register

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestInputParameters(t *testing.T) {
	require := require.New(t)

	var executor *RegistrationExecutor

	cmd := &cobra.Command{}
	SetupRegisterClusterCmd(cmd, genericiooptions.NewTestIOStreamsDiscard())
	cmd.Commands()[0].RunE = func(cmd *cobra.Command, args []string) error {
		executor = NewRegistrationExecutorFromRegisterClusterCmd(*cmd)
		return nil
	}
	cmd.Root().SetArgs([]string{
		"register",
		"--source-context", "source-ctx",
		"--source-kubeconfig", "testsourcekubeconfig",
		"--dest-kubeconfig", "testdestkubeconfig",
		"--dest-context", "dest-ctx",
		"--source-namespace", "source-namespace",
		"--dest-namespace", "dest-namespace",
		"--serviceaccount-name", "test-sa",
		"--override-src-ip", "127.0.0.2",
		"--override-src-port", "9999"})

	require.NoError(cmd.Execute())

	require.Equal("127.0.0.2", executor.OverrideSourceIP)
	require.Equal("9999", executor.OverrideSourcePort)
	require.Equal("testsourcekubeconfig", executor.SourceKubeconfig)
	require.Equal("source-ctx", executor.SourceContext)
	require.Equal("testdestkubeconfig", executor.DestKubeconfig)
	require.Equal("dest-ctx", executor.DestContext)
	require.Equal("source-namespace", executor.SourceNamespace)
	require.Equal("dest-namespace", executor.DestNamespace)
	require.Equal("test-sa", executor.ServiceAccount)
}

func TestIncorrectParameters(t *testing.T) {
	require := require.New(t)

	cmd := &cobra.Command{}
	cmd.SilenceUsage = true
	SetupRegisterClusterCmd(cmd, genericiooptions.NewTestIOStreamsDiscard())
	cmd.Commands()[0].RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	cmd.Root().SetArgs([]string{
		"register",
		"--service-account", "test-sa",
	})

	err := cmd.Execute()
	require.Error(err)
	require.Equal("unknown flag: --service-account", err.Error())
}
