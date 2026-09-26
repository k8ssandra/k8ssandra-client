package kubernetes

import (
	"math/rand"
	"os/user"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
)

func TestLoadConfigurationContext(t *testing.T) {
	require := require.New(t)

	conf, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	require.NoError(err)

	if _, found := conf.Contexts["kind-kind"]; !found {
		t.Skip("kind-kind context not found in kubeconfig")
	}

	restConfig, err := LoadClientConfig(defaultConfigPath(t), "kind-kind")
	require.NoError(err)

	// mTLS auth has KeyData & CertData
	require.NotNil(restConfig.KeyData)
	require.NotNil(restConfig.CertData)
	require.NotNil(restConfig.CAData)
	require.True(strings.HasPrefix(restConfig.Host, "https://127.0.0.1"))
}

func TestVerifyContextWasChanged(t *testing.T) {
	require := require.New(t)

	conf, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	require.NoError(err)

	if len(conf.Contexts) < 1 {
		t.Skip("no context found in kubeconfig")
	}

	keys := reflect.ValueOf(conf.Contexts).MapKeys()
	randomContextName := keys[rand.Intn(len(keys))].String()

	randomContext := conf.Contexts[randomContextName]
	randomServer := conf.Clusters[randomContext.Cluster].Server

	restConfig, err := LoadClientConfig(defaultConfigPath(t), randomContextName)
	require.NoError(err)
	require.NotNil(restConfig.CAData)
	require.Equal(randomServer, restConfig.Host)
}

func defaultConfigPath(t *testing.T) string {
	u, err := user.Current()
	require.NoError(t, err)
	return filepath.Join(u.HomeDir, ".kube", "config")
}
