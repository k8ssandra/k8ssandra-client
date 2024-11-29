package helmutil_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k8ssandra/k8ssandra-client/pkg/helmutil"
	"github.com/k8ssandra/k8ssandra-client/pkg/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestUpgradingCRDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require := require.New(t)
	chartNames := []string{"cass-operator"}
	for _, chartName := range chartNames {
		namespace := env.CreateNamespace(t)
		kubeClient := env.GetClientInNamespace(namespace)
		require.NoError(cleanCache("k8ssandra", chartName))

		// creating new upgrader
		u, err := helmutil.NewUpgrader(kubeClient, helmutil.K8ssandraRepoName, helmutil.StableK8ssandraRepoURL, chartName, []string{})
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
		}, time.Minute*1, time.Millisecond*100)

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

func TestUpgradingStoredVersions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	require := require.New(t)
	chartName := "test-chart"
	namespace := env.CreateNamespace(t)
	kubeClient := env.GetClientInNamespace(namespace)
	require.NoError(cleanCache("k8ssandra", chartName))

	// Copy testfiles
	chartDir, err := helmutil.GetChartTargetDir(helmutil.K8ssandraRepoName, chartName)
	require.NoError(err)

	crdDir := filepath.Join(chartDir, "crds")
	_, err = util.CreateIfNotExistsDir(crdDir)
	require.NoError(err)
	crdSrc := filepath.Join("..", "..", "testfiles", "crd-upgrader", "multiversion-clientconfig-mockup-v1alpha1.yaml")
	require.NoError(copyFile(crdSrc, filepath.Join(crdDir, "clientconfig.yaml")))

	testOptions := envtest.CRDInstallOptions{
		PollInterval: 100 * time.Millisecond,
		MaxTime:      10 * time.Second,
	}

	// creating new upgrader
	u, err := helmutil.NewUpgrader(kubeClient, helmutil.K8ssandraRepoName, helmutil.StableK8ssandraRepoURL, chartName, []string{})
	require.NoError(err)

	crds, err := u.Upgrade(context.TODO(), "0.1.0")
	require.NoError(err)

	targetCrd := &apiextensions.CustomResourceDefinition{}
	objs := []*apiextensions.CustomResourceDefinition{}
	for _, crd := range crds {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), targetCrd)
		require.NoError(err)
		objs = append(objs, targetCrd)
	}

	require.NotEmpty(objs)
	require.NotEmpty(targetCrd.GetName())
	require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
	require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: targetCrd.GetName()}, targetCrd))

	require.Equal([]string{"v1alpha1"}, targetCrd.Status.StoredVersions)

	// Upgrade to 0.2.0

	require.NoError(cleanCache("k8ssandra", chartName))
	_, err = util.CreateIfNotExistsDir(crdDir)
	require.NoError(err)
	crdSrc = filepath.Join("..", "..", "testfiles", "crd-upgrader", "multiversion-clientconfig-mockup-both.yaml")
	require.NoError(copyFile(crdSrc, filepath.Join(crdDir, "clientconfig.yaml")))

	crds, err = u.Upgrade(context.TODO(), "0.2.0")
	require.NoError(err)
	for _, crd := range crds {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), targetCrd)
		require.NoError(err)
		objs = append(objs, targetCrd)
	}
	require.NotEmpty(objs)
	require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
	require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: targetCrd.GetName()}, targetCrd))
	require.Equal([]string{"v1alpha1", "v1beta1"}, targetCrd.Status.StoredVersions)

	// Upgrade to 0.3.0

	require.NoError(cleanCache("k8ssandra", chartName))
	_, err = util.CreateIfNotExistsDir(crdDir)
	require.NoError(err)
	crdSrc = filepath.Join("..", "..", "testfiles", "crd-upgrader", "multiversion-clientconfig-mockup-v1beta1.yaml")
	require.NoError(copyFile(crdSrc, filepath.Join(crdDir, "clientconfig.yaml")))

	crds, err = u.Upgrade(context.TODO(), "0.3.0")
	require.NoError(err)
	for _, crd := range crds {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), targetCrd)
		require.NoError(err)
		objs = append(objs, targetCrd)
	}
	require.NotEmpty(objs)
	require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
	require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: targetCrd.GetName()}, targetCrd))
	require.Equal([]string{"v1beta1"}, targetCrd.Status.StoredVersions)

	// Sanity check, install 0.2.0 and only update to 0.3.0 (there should be no storedVersion of v1alpha1)
	require.NoError(kubeClient.Delete(context.TODO(), targetCrd))
	require.Eventually(func() bool {
		err = kubeClient.Get(context.TODO(), client.ObjectKey{Name: targetCrd.GetName()}, targetCrd)
		return err != nil && client.IgnoreNotFound(err) == nil
	}, time.Second*5, time.Millisecond*100)

	// Install 0.2.0

	require.NoError(cleanCache("k8ssandra", chartName))
	_, err = util.CreateIfNotExistsDir(crdDir)
	require.NoError(err)
	crdSrc = filepath.Join("..", "..", "testfiles", "crd-upgrader", "multiversion-clientconfig-mockup-both.yaml")
	require.NoError(copyFile(crdSrc, filepath.Join(crdDir, "clientconfig.yaml")))

	crds, err = u.Upgrade(context.TODO(), "0.2.0")
	require.NoError(err)
	for _, crd := range crds {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), targetCrd)
		require.NoError(err)
		objs = append(objs, targetCrd)
	}
	require.NotEmpty(objs)
	require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
	require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: targetCrd.GetName()}, targetCrd))
	require.Equal([]string{"v1beta1"}, targetCrd.Status.StoredVersions)

	// Upgrade to 0.3.0

	require.NoError(cleanCache("k8ssandra", chartName))
	_, err = util.CreateIfNotExistsDir(crdDir)
	require.NoError(err)
	crdSrc = filepath.Join("..", "..", "testfiles", "crd-upgrader", "multiversion-clientconfig-mockup-v1beta1.yaml")
	require.NoError(copyFile(crdSrc, filepath.Join(crdDir, "clientconfig.yaml")))

	crds, err = u.Upgrade(context.TODO(), "0.3.0")
	require.NoError(err)
	for _, crd := range crds {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), targetCrd)
		require.NoError(err)
		objs = append(objs, targetCrd)
	}
	require.NotEmpty(objs)
	require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
	require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: targetCrd.GetName()}, targetCrd))
	require.Equal([]string{"v1beta1"}, targetCrd.Status.StoredVersions)
}

func copyFile(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to open %s", source))
	}
	defer src.Close()

	dst, err := os.Create(target)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to open %s", target))
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
