package helmutil

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/k8ssandra/k8ssandra-client/pkg/util"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

// DownloadChartRelease fetches the k8ssandra target version and extracts it to a directory which path is returned
func DownloadChartRelease(repoName, repoURL, chartName, chartVersion string, options ...getter.Option) (string, error) {
	// Unfortunately, the helm's chart pull command uses "internal" marked structs, so it can't be used for
	// pulling the data. Thus, we need to replicate the implementation here and use our own cache
	settings := cli.New()
	var out strings.Builder

	c := downloader.ChartDownloader{
		Out: &out,
		// Keyring: p.Keyring,
		Verify:  downloader.VerifyNever,
		Getters: getter.All(settings),
		Options: []getter.Option{
			// getter.WithBasicAuth(p.Username, p.Password),
			// getter.WithTLSClientConfig(p.CertFile, p.KeyFile, p.CaFile),
			// getter.WithInsecureSkipVerifyTLS(p.InsecureSkipTLSverify),
		},
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}

	c.Options = append(c.Options, options...)

	// helm repo add k8ssandra https://helm.k8ssandra.io/
	r, err := repo.NewChartRepository(&repo.Entry{
		Name: repoName,
		URL:  repoURL,
	}, getter.All(settings))

	if err != nil {
		return "", err
	}

	// helm repo update k8ssandra
	index, err := r.DownloadIndexFile()
	if err != nil {
		return "", err
	}

	// Read the index file for the repository to get chart information and return chart URL
	repoIndex, err := repo.LoadIndexFile(index)
	if err != nil {
		return "", err
	}

	// chart name, chart version
	cv, err := repoIndex.Get(chartName, chartVersion)
	if err != nil {
		return "", err
	}

	url, err := repo.ResolveReferenceURL(repoURL, cv.URLs[0])
	if err != nil {
		return "", err
	}

	// Download to filesystem for extraction purposes
	dir, err := os.MkdirTemp("", "helmutil-")
	if err != nil {
		return "", err
	}

	// _ is ProvenanceVerify (TODO we might want to verify the release)
	saved, _, err := c.DownloadTo(url, chartVersion, dir)
	if err != nil {
		return "", err
	}

	return saved, nil
}

func ExtractChartRelease(saved, repoName, chartName, chartVersion string) (string, error) {
	// Extract the files
	subDir := filepath.Join(chartName, chartVersion)
	extractDir, err := util.GetCacheDir(repoName, subDir)
	if err != nil {
		return "", err
	}

	if _, err := util.CreateIfNotExistsDir(extractDir); err != nil {
		return "", err
	}

	// extractDir is a target directory
	err = chartutil.ExpandFile(extractDir, saved)
	if err != nil {
		return "", err
	}

	return extractDir, nil
}

func GetChartTargetDir(repoName, chartName, chartVersion string) (string, error) {
	extractDir, err := util.GetCacheDir(repoName, chartName)
	if err != nil {
		return "", err
	}

	extractDir = filepath.Join(extractDir, chartVersion)

	return extractDir, err
}

func Release(cfg *action.Configuration, releaseName string) (*release.Release, error) {
	getAction := action.NewGet(cfg)
	return getAction.Run(releaseName)
}

func ListInstallations(cfg *action.Configuration) ([]*release.Release, error) {
	listAction := action.NewList(cfg)
	listAction.AllNamespaces = true
	return listAction.Run()
}

func Install(cfg *action.Configuration, releaseName, path, namespace string, values map[string]interface{}, devel bool, skipCRDs bool, timeout time.Duration) (*release.Release, error) {
	installAction := action.NewInstall(cfg)
	installAction.ReleaseName = releaseName
	installAction.Namespace = namespace
	installAction.CreateNamespace = true
	installAction.Atomic = true
	installAction.Wait = true
	if timeout > 0 {
		installAction.Timeout = timeout
	}
	if skipCRDs {
		installAction.SkipCRDs = true
	}
	if devel {
		installAction.Devel = true
		installAction.Version = ">0.0.0.0"
	}
	chartReq, err := loader.Load(path)
	if err != nil {
		return nil, err
	}

	return installAction.Run(chartReq, values)
}

func Uninstall(cfg *action.Configuration, releaseName string) (*release.UninstallReleaseResponse, error) {
	uninstallAction := action.NewUninstall(cfg)
	return uninstallAction.Run(releaseName)
}

// ValuesYaml fetches the chartVersion's values.yaml file for editing purposes
func ValuesYaml(chartVersion string) (io.Reader, error) {
	return nil, nil
}
