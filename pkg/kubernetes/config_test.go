package kubernetes

import (
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigurationContext(t *testing.T) {
	t.Skip("requires kind cluster with defaults to be running")
	require := require.New(t)

	u, err := user.Current()
	require.NoError(err)
	cfgFile := filepath.Join(u.HomeDir, ".kube", "config")

	restConfig, err := LoadClientConfig(cfgFile, "kind-kind")
	require.NoError(err)

	require.NotNil(restConfig.KeyData)
	require.NotNil(restConfig.CertData)
	require.True(strings.HasPrefix(restConfig.Host, "https://127.0.0.1"))
}
