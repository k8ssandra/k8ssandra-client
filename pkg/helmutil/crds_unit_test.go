package helmutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindCRDDirs(t *testing.T) {
	require := require.New(t)
	chartDir, err := os.MkdirTemp("", "k8ssandra")
	defer os.RemoveAll(chartDir)
	require.NoError(err)

	require.NoError(os.MkdirAll(chartDir+"/downstream-operator/crds", 0755))

	dirs, err := findCRDDirs(chartDir)
	require.NoError(err)

	require.Len(dirs, 1)
	require.Equal(chartDir+"/downstream-operator/crds", dirs[0])

	require.NoError(os.MkdirAll(chartDir+"/downstream-operator/charts/k8ssandra-operator/crds", 0755))
	require.NoError(os.MkdirAll(chartDir+"/downstream-operator/charts/k8ssandra-operator/charts/cass-operator/crds", 0755))

	dirs, err = findCRDDirs(chartDir)
	require.NoError(err)

	require.Len(dirs, 3)
	require.Contains(dirs, chartDir+"/downstream-operator/crds")
	require.Contains(dirs, chartDir+"/downstream-operator/charts/k8ssandra-operator/crds")
	require.Contains(dirs, chartDir+"/downstream-operator/charts/k8ssandra-operator/charts/cass-operator/crds")
}
