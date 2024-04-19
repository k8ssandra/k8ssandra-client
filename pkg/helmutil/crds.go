package helmutil

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	deser "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
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
func (u *Upgrader) Upgrade(ctx context.Context, chartVersion string) ([]unstructured.Unstructured, error) {
	log.SetLevel(log.DebugLevel)
	log.Info("Processing request to upgrade project CustomResourceDefinitions", "repoName", u.repoName, "chartName", u.chartName, "chartVersion", chartVersion)
	chartDir, err := GetChartTargetDir(u.repoName, u.chartName)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(chartDir); os.IsNotExist(err) {
		log.Info("Downloading chart release from remote repository", "repoURL", u.repoURL, "chartName", u.chartName, "chartVersion", chartVersion)
		downloadDir, err := DownloadChartRelease(u.repoName, u.repoURL, u.chartName, chartVersion)
		if err != nil {
			return nil, err
		}

		extractDir, err := ExtractChartRelease(downloadDir, u.repoName, u.chartName, chartVersion)
		if err != nil {
			return nil, err
		}
		chartDir = extractDir
	} else {
		log.Info("Using cached chart release", "directory", chartDir)
	}

	crds := make([]unstructured.Unstructured, 0)

	// For each dir under the charts subdir, check the "crds/"
	paths, _ := findCRDDirs(chartDir)

	for _, path := range paths {
		log.Debug("Processing CustomResourceDefinition directory", "path", path)
		err = parseChartCRDs(&crds, path)
		if err != nil {
			return nil, err
		}
	}

	for _, obj := range crds {
		log.Info("Processing CustomResourceDefinition", "name", obj.GetName())
		existingCrd := obj.DeepCopy()
		err = u.client.Get(ctx, client.ObjectKey{Name: obj.GetName()}, existingCrd)
		if apierrors.IsNotFound(err) {
			log.Debug("Creating CustomResourceDefinition", "name", obj.GetName())
			if err = u.client.Create(ctx, &obj); err != nil {
				return nil, errors.Wrapf(err, "failed to create CRD %s", obj.GetName())
			}
		} else if err != nil {
			return nil, errors.Wrapf(err, "failed to fetch state of %s", obj.GetName())
		} else {
			log.Debug("Updating CustomResourceDefinition", "name", obj.GetName())
			obj.SetResourceVersion(existingCrd.GetResourceVersion())
			if err = u.client.Update(ctx, &obj); err != nil {
				return nil, errors.Wrapf(err, "failed to update CRD %s", obj.GetName())
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
		log.Debug("Parsing CustomResourceDefinition file", "path", path)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if len(b) == 0 {
			return nil
		}

		reader := k8syaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(b)))
		doc, err := reader.Read()
		if err != nil {
			return err
		}

		crd := unstructured.Unstructured{}

		dec := deser.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		_, gvk, err := dec.Decode(doc, nil, &crd)
		if err != nil {
			log.Error("Failed to decode CustomResourceDefinition", "path", path, "error", err)
			return nil
		}

		if gvk.Kind != "CustomResourceDefinition" {
			log.Error("File is not a CustomResourceDefinition", "path", path, "kind", gvk.Kind)
			return nil
		}

		*crds = append(*crds, crd)

		return nil
	})

	return errOuter
}
