package helmutil_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/k8ssandra/k8ssandra-client/pkg/helmutil"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestUpgradingCRDs(t *testing.T) {
	require := require.New(t)
	chartNames := []string{"cass-operator"}
	for _, chartName := range chartNames {
		namespace := env.CreateNamespace(t)
		kubeClient := env.GetClientInNamespace(namespace)
		require.NoError(cleanCache("k8ssandra", chartName))

		// creating new upgrader
		u, err := helmutil.NewUpgrader(kubeClient, helmutil.K8ssandraRepoName, helmutil.StableK8ssandraRepoURL, chartName)
		require.NoError(err)

		crds, err := u.Upgrade(context.TODO(), "0.42.0")
		require.NoError(err)

		testOptions := envtest.CRDInstallOptions{
			PollInterval: 100 * time.Millisecond,
			MaxTime:      10 * time.Second,
		}

		cassDCCRD := &apiextensions.CustomResourceDefinition{}
		objs := []*apiextensions.CustomResourceDefinition{}
		for _, crd := range crds {
			if crd.GetName() == "cassandradatacenters.cassandra.datastax.com" {
				err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), cassDCCRD)
				require.NoError(err)
			}
			objs = append(objs, cassDCCRD)
		}

		require.NotEmpty(objs)
		require.NotEmpty(cassDCCRD.GetName())
		require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
		require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: cassDCCRD.GetName()}, cassDCCRD))
		ver := cassDCCRD.GetResourceVersion()

		descRunsAsCassandra := cassDCCRD.Spec.Versions[0].DeepCopy().Schema.OpenAPIV3Schema.Properties["spec"].Properties["dockerImageRunsAsCassandra"].Description
		require.False(strings.HasPrefix(descRunsAsCassandra, "DEPRECATED"))

		// Upgrading to 0.46.1
		require.NoError(cleanCache("k8ssandra", chartName))
		_, err = u.Upgrade(context.TODO(), "0.46.1")
		require.NoError(err)

		require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
		require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: cassDCCRD.GetName()}, cassDCCRD))

		require.Eventually(func() bool {
			require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: cassDCCRD.GetName()}, cassDCCRD))
			newver := cassDCCRD.GetResourceVersion()
			return newver != ver
		}, time.Minute*1, time.Second*5)

		descRunsAsCassandra = cassDCCRD.Spec.Versions[0].DeepCopy().Schema.OpenAPIV3Schema.Properties["spec"].Properties["dockerImageRunsAsCassandra"].Description
		require.True(strings.HasPrefix(descRunsAsCassandra, "DEPRECATED"))
	}
}

func cleanCache(repoName, chartName string) error {
	chartDir, err := helmutil.GetChartTargetDir(repoName, chartName)
	if err != nil {
		return err
	}

	return os.RemoveAll(chartDir)
}
