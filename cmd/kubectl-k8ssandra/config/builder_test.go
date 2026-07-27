package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestDefaultBuilderCommand(t *testing.T) {
	require := require.New(t)

	options := newBuilderOptions(genericiooptions.NewTestIOStreamsDiscard())
	cmd := newBuilderCmd(options)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	cmd.Root().SetArgs([]string{"build"})
	require.NoError(cmd.Execute())
	require.Empty(options.inputDir)
	require.Empty(options.outputDir)
	require.False(options.sidecar)
	require.Empty(options.configBuilderOptions)
}

func TestSidecarBuilderCommand(t *testing.T) {
	require := require.New(t)

	options := newBuilderOptions(genericiooptions.NewTestIOStreamsDiscard())
	cmd := newBuilderCmd(options)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	cmd.Root().SetArgs([]string{"build", "--sidecar", "--input", "/input-dir", "--output", "/output-dir"})
	require.NoError(cmd.Execute())
	require.Equal("/input-dir", options.inputDir)
	require.Equal("/output-dir", options.outputDir)
	require.True(options.sidecar)
	require.Len(options.configBuilderOptions, 1)
}

func TestSidecarBuilderCommandMissingDirectories(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "both",
			args: []string{"build", "--sidecar"},
		},
		{
			name: "input",
			args: []string{"build", "--sidecar", "--output", "/output-dir"},
		},
		{
			name: "output",
			args: []string{"build", "--sidecar", "--input", "/input-dir"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := NewBuilderCmd(genericiooptions.NewTestIOStreamsDiscard())
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				return nil
			}

			cmd.Root().SetArgs(test.args)
			require.Error(t, cmd.Execute())
		})
	}
}
