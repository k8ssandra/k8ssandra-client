package helmutil

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v4/pkg/action"
	chart "helm.sh/helm/v4/pkg/chart/v2"
	kubefake "helm.sh/helm/v4/pkg/kube/fake"
	"helm.sh/helm/v4/pkg/release/common"
	release "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"
)

func helmConfiguration(t *testing.T, releases ...*release.Release) *action.Configuration {
	t.Helper()

	memory := driver.NewMemory()
	cfg := action.NewConfiguration()
	cfg.KubeClient = &kubefake.PrintingKubeClient{Out: io.Discard}
	cfg.Releases = storage.Init(memory)
	for _, rel := range releases {
		require.NoError(t, cfg.Releases.Create(rel))
	}
	// Search all namespaces in memory to test namespace filtering
	memory.SetNamespace("")
	return cfg
}

func helmRelease(namespace, releaseName, chartName string, status common.Status, revision int) *release.Release {
	return &release.Release{
		Name:      releaseName,
		Namespace: namespace,
		Version:   revision,
		Info:      &release.Info{Status: status},
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{Name: chartName},
		},
	}
}

func TestGetDeployedReleaseName(t *testing.T) {
	tests := []struct {
		name            string
		releases        []*release.Release
		queryNamespace  string
		chartNamePrefix string
		expectedRelease string
		expectedError   string
	}{
		{
			name:            "returns release name",
			releases:        []*release.Release{helmRelease("k8ssandra-operator", "k8ssandra-operator", "k8ssandra-operator", common.StatusDeployed, 1)},
			queryNamespace:  "k8ssandra-operator",
			expectedRelease: "k8ssandra-operator",
		},
		{
			name:            "returns custom release name",
			releases:        []*release.Release{helmRelease("my-ns", "my-release", "custom-chart", common.StatusDeployed, 1)},
			queryNamespace:  "my-ns",
			expectedRelease: "my-release",
		},
		{
			name: "lets helm select deployed releases and latest revisions",
			releases: []*release.Release{
				helmRelease("k8ssandra-operator", "current-release", "k8ssandra-operator", common.StatusSuperseded, 1),
				helmRelease("k8ssandra-operator", "current-release", "k8ssandra-operator", common.StatusDeployed, 2),
				helmRelease("k8ssandra-operator", "failed-release", "k8ssandra-operator", common.StatusFailed, 1),
				helmRelease("other-ns", "other-release", "k8ssandra-operator", common.StatusDeployed, 1),
			},
			queryNamespace:  "k8ssandra-operator",
			expectedRelease: "current-release",
		},
		{
			name: "filters by chart name prefix",
			releases: []*release.Release{
				helmRelease("k8ssandra-operator", "custom-release", "k8ssandra-operator", common.StatusDeployed, 1),
				helmRelease("k8ssandra-operator", "cert-release", "cert-manager", common.StatusDeployed, 1),
			},
			queryNamespace:  "k8ssandra-operator",
			chartNamePrefix: "mission-",
			expectedRelease: "custom-release",
		},
		{
			name:           "returns error when no release is deployed in namespace",
			releases:       []*release.Release{helmRelease("k8ssandra-operator", "old-release", "k8ssandra-operator", common.StatusSuperseded, 1)},
			queryNamespace: "k8ssandra-operator",
			expectedError:  `no deployed Helm release found in namespace "k8ssandra-operator"`,
		},
		{
			name: "returns error when multiple releases are deployed",
			releases: []*release.Release{
				helmRelease("k8ssandra-operator", "release-b", "chart-b", common.StatusDeployed, 1),
				helmRelease("k8ssandra-operator", "release-a", "chart-a", common.StatusDeployed, 1),
			},
			queryNamespace: "k8ssandra-operator",
			expectedError:  `multiple deployed Helm releases found in namespace "k8ssandra-operator": [release-a release-b]`,
		},
		{
			name: "returns error when no chart name matches prefix",
			releases: []*release.Release{
				helmRelease("k8ssandra-operator", "cert-release", "cert-manager", common.StatusDeployed, 1),
			},
			queryNamespace:  "k8ssandra-operator",
			chartNamePrefix: "mission-",
			expectedError:   `no deployed Helm release with chart name prefix "mission-" found in namespace "k8ssandra-operator"`,
		},
		{
			name: "returns error when chart name prefix is ambiguous",
			releases: []*release.Release{
				helmRelease("k8ssandra-operator", "release-b", "k8ssandra-operator-crds", common.StatusDeployed, 1),
				helmRelease("k8ssandra-operator", "release-a", "k8ssandra-operator", common.StatusDeployed, 1),
			},
			queryNamespace:  "k8ssandra-operator",
			chartNamePrefix: "k8ssandra-operator",
			expectedError:   `multiple deployed Helm releases with chart name prefix "k8ssandra-operator" found in namespace "k8ssandra-operator": [release-a release-b]`,
		},
		{
			name:          "returns error when namespace is empty",
			expectedError: "namespace must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := helmConfiguration(t, tt.releases...)
			name, err := DeployedReleaseName(cfg, tt.queryNamespace, tt.chartNamePrefix)
			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				assert.Empty(t, name)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRelease, name)
			}
		})
	}

	t.Run("wraps helm list errors", func(t *testing.T) {
		listErr := errors.New("connection refused")
		cfg := helmConfiguration(t)
		cfg.KubeClient = &kubefake.FailingKubeClient{
			PrintingKubeClient: kubefake.PrintingKubeClient{Out: io.Discard},
			ConnectionError:    listErr,
		}

		name, err := DeployedReleaseName(cfg, "k8ssandra-operator", "")
		require.ErrorIs(t, err, listErr)
		assert.Empty(t, name)
		assert.EqualError(t, err, `list deployed Helm releases in namespace "k8ssandra-operator": connection refused`)
	})

	t.Run("ignores releases without chart metadata", func(t *testing.T) {
		withoutMetadata := helmRelease("k8ssandra-operator", "release-without-metadata", "", common.StatusDeployed, 1)
		withoutMetadata.Chart = nil
		matching := helmRelease("k8ssandra-operator", "matching-release", "k8ssandra-operator", common.StatusDeployed, 1)

		name, err := DeployedReleaseName(
			helmConfiguration(t, withoutMetadata, matching),
			"k8ssandra-operator",
			"mission-",
		)
		require.NoError(t, err)
		assert.Equal(t, "matching-release", name)
	})
}
