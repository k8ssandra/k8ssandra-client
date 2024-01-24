package helmutil_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/k8ssandra/k8ssandra-client/pkg/helmutil"
	"github.com/stretchr/testify/require"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	// +kubebuilder:scaffold:imports
)

func TestUpgradingCRDs(t *testing.T) {
	require := require.New(t)
	chartNames := []string{"k8ssandra"}
	for _, chartName := range chartNames {
		t.Run(fmt.Sprintf("CRD upgrade for chart name %s", chartName), func(t *testing.T) {
			namespace := env.CreateNamespace(t)
			kubeClient := env.Client(namespace)

			// creating new upgrader
			u, err := helmutil.NewUpgrader(kubeClient, helmutil.K8ssandraRepoName, helmutil.StableK8ssandraRepoURL, chartName)
			require.NoError(err)

			// Upgrading / installing 1.0.0
			var crds []unstructured.Unstructured
			require.Eventually(func() bool {
				_, err := u.Upgrade(context.TODO(), "1.0.0")
				return err == nil
			}, time.Minute*1, time.Second*5)

			testOptions := envtest.CRDInstallOptions{
				PollInterval: 100 * time.Millisecond,
				MaxTime:      10 * time.Second,
			}

			unstructuredCRD := &unstructured.Unstructured{}
			cassDCCRD := &apiextensions.CustomResourceDefinition{}
			objs := []*apiextensions.CustomResourceDefinition{}
			for _, crd := range crds {
				if crd.GetName() == "cassandradatacenters.cassandra.datastax.com" {
					unstructuredCRD = crd.DeepCopy()
					err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), cassDCCRD)
					require.NoError(err)
				}
				objs = append(objs, cassDCCRD)
			}

			require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
			require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: cassDCCRD.GetName()}, cassDCCRD))
			ver := cassDCCRD.GetResourceVersion()

			_, found, err := unstructured.NestedFieldNoCopy(unstructuredCRD.Object, "spec", "validation", "openAPIV3Schema", "properties", "spec", "properties", "configSecret")
			require.NoError(err)
			require.False(found)

			// Upgrading to 1.5.1
			crds, err = u.Upgrade(context.TODO(), "1.5.1")
			require.NoError(err)

			objs = []*apiextensions.CustomResourceDefinition{}
			for _, crd := range crds {
				if crd.GetName() == "cassandradatacenters.cassandra.datastax.com" {
					unstructuredCRD = crd.DeepCopy()
					err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), cassDCCRD)
					require.NoError(err)
					objs = append(objs, cassDCCRD)
				}
			}

			require.NoError(envtest.WaitForCRDs(env.RestConfig(), objs, testOptions))
			require.NoError(kubeClient.Get(context.TODO(), client.ObjectKey{Name: cassDCCRD.GetName()}, cassDCCRD))

			require.Eventually(func() bool {
				newver := cassDCCRD.GetResourceVersion()
				return newver == ver
			}, time.Minute*1, time.Second*5)

			versionsSlice, found, err := unstructured.NestedSlice(unstructuredCRD.Object, "spec", "versions")
			require.NoError(err)
			require.True(found)

			_, found, err = unstructured.NestedFieldNoCopy(versionsSlice[0].(map[string]interface{}), "schema", "openAPIV3Schema", "properties", "spec", "properties", "configSecret")
			require.NoError(err)
			require.True(found)
		})
	}
}
