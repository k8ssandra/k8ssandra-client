package helmutil

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Upgrader is a utility to update the CRDs in a helm chart's pre-upgrade hook
type Upgrader struct {
	client    client.Client
	repoName  string
	repoURL   string
	chartName string
}

// NewUpgrader returns a new Upgrader client
func NewUpgrader(c client.Client, repoName, repoURL, chartName string) (*Upgrader, error) {
	return &Upgrader{
		client:    c,
		repoName:  repoName,
		repoURL:   repoURL,
		chartName: chartName,
	}, nil
}

// Upgrade installs the missing CRDs or updates them if they exists already
func (u *Upgrader) Upgrade(ctx context.Context, targetVersion string) ([]unstructured.Unstructured, error) {
	extractDir, err := DownloadChartRelease(u.repoName, u.repoURL, u.chartName, targetVersion)
	if err != nil {
		return nil, err
	}

	// reaper and medusa subdirs have the required yaml files
	chartPath := filepath.Join(extractDir, u.repoName)
	defer os.RemoveAll(chartPath)

	crds := make([]unstructured.Unstructured, 0)

	// For each dir under the charts subdir, check the "crds/"
	paths, _ := findCRDDirs(chartPath)

	for _, path := range paths {
		err = parseChartCRDs(&crds, path)
		if err != nil {
			return nil, err
		}
	}

	var res []client.Object
	for _, obj := range crds {
		res = append(res, &obj)
	}

	for _, obj := range res {
		if err := u.client.Create(context.TODO(), obj); err != nil {
			if apierrors.IsAlreadyExists(err) {
				if err := u.client.Update(context.TODO(), obj); err != nil {
					return nil, err
				}
			}
		}
	}

	return crds, err
}

func findCRDDirs(chartDir string) ([]string, error) {
	dirs := make([]string, 0)
	err := filepath.Walk(chartDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasSuffix(path, "crds") {
				dirs = append(dirs, path)
			}
			return nil
		}
		return nil
	})
	return dirs, err
}

func parseChartCRDs(crds *[]unstructured.Unstructured, crdDir string) error {
	errOuter := filepath.Walk(crdDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Add to CRDs ..
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		reader := k8syaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(b)))
		doc, err := reader.Read()
		if err != nil {
			return err
		}

		crd := unstructured.Unstructured{}

		if err = yaml.Unmarshal(doc, &crd); err != nil {
			return err
		}

		*crds = append(*crds, crd)
		return nil
	})

	return errOuter
}
