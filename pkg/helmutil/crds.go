package helmutil

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	deser "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AllSubCharts = "_"
)

// Upgrader is a utility to update the CRDs in a helm chart's pre-upgrade hook
type Upgrader struct {
	client    client.Client
	repoName  string
	repoURL   string
	chartName string
	subCharts []string
}

// NewUpgrader returns a new Upgrader client
func NewUpgrader(c client.Client, repoName, repoURL, chartName string, subCharts []string) (*Upgrader, error) {
	return &Upgrader{
		client:    c,
		repoName:  repoName,
		repoURL:   repoURL,
		chartName: chartName,
		subCharts: subCharts,
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

	if fs, err := os.Stat(chartDir); os.IsNotExist(err) {
		log.Info("Downloading chart release from remote repository", "repoURL", u.repoURL, "chartName", u.chartName, "chartVersion", chartVersion, "chartDir", chartDir)
		downloadDir, err := DownloadChartRelease(u.repoName, u.repoURL, u.chartName, chartVersion)
		if err != nil {
			return nil, err
		}

		extractDir, err := ExtractChartRelease(downloadDir, u.repoName, u.chartName, chartVersion)
		if err != nil {
			return nil, err
		}
		chartDir = extractDir
	} else if err != nil {
		log.Error("Failed to check chart release directory", "error", err)
		return nil, err
	} else if !fs.IsDir() {
		err := fmt.Errorf("chart release is not a directory: %s", chartDir)
		log.Error("Target chart release path is not a directory", "directory", chartDir, "error", err)
		return nil, err
	} else {
		log.Info("Using cached chart release", "directory", chartDir)
	}

	crds := make([]unstructured.Unstructured, 0)

	// For each dir under the charts subdir, check the "crds/"
	paths, _ := findCRDDirs(chartDir, u.subCharts)

	for _, path := range paths {
		log.Debug("Processing CustomResourceDefinition directory", "path", path)
		if err := parseChartCRDs(&crds, path); err != nil {
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

			// TODO We need to check which versions we have available here before updating
			unstructured := obj.UnstructuredContent()
			var definition apiextensionsv1.CustomResourceDefinition
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, &definition); err != nil {
				return nil, errors.Wrapf(err, "failed to convert unstructured to CustomResourceDefinition %s", obj.GetName())
			}

			updatedVersions := make([]string, 0, len(definition.Spec.Versions))
			for _, version := range definition.Spec.Versions {
				updatedVersions = append(updatedVersions, version.Name)
			}
			log.Debug("Read CustomResourceDefinition versions", "name", obj.GetName(), "versions", updatedVersions)

			existing := existingCrd.UnstructuredContent()
			var existingDefinition apiextensionsv1.CustomResourceDefinition
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(existing, &existingDefinition); err != nil {
				return nil, errors.Wrapf(err, "failed to convert unstructured to CustomResourceDefinition %s", obj.GetName())
			}

			storedVersions := existingDefinition.Status.StoredVersions

			if !slices.Equal(storedVersions, updatedVersions) {

				// Check if storedVersion has any versions that are not in updatedVersions
				// If so, we need to remove them from the storedVersions
				removed := false
				for _, storedVersion := range storedVersions {
					if !slices.Contains(updatedVersions, storedVersion) {
						log.Debug("Removing CustomResourceDefinition version", "name", obj.GetName(), "version", storedVersion)
						// storedVersions = slices.DeleteFunc(storedVersions, func(e string) bool { return e == storedVersion })
						removed = true
					}
				}

				if removed {
					log.Debug("Updating CustomResourceDefinition versions", "name", obj.GetName(), "storedVersions", storedVersions, "updatedVersions", updatedVersions)
					existingDefinition.Status.StoredVersions = updatedVersions
					if err := u.client.Status().Update(ctx, &existingDefinition); err != nil {
						return nil, errors.Wrapf(err, "failed to update CRD storedVersions %s", obj.GetName())
					}
					obj.SetResourceVersion(existingDefinition.GetResourceVersion())
				}
			}

			if err = u.client.Update(ctx, &obj); err != nil {
				return nil, errors.Wrapf(err, "failed to update CRD %s", obj.GetName())
			}
		}
	}

	return crds, err
}

func findCRDDirs(chartDir string, subCharts []string) ([]string, error) {
	chartsList := make(map[string]struct{})
	for _, chart := range subCharts {
		chartsList[chart] = struct{}{}
	}

	chartFilter := func(path string, info os.FileInfo) bool {
		if !info.IsDir() || filepath.Base(path) != "crds" {
			return false
		}

		chartParts := strings.Split(filepath.Clean(path), string(os.PathSeparator))
		chartName := chartParts[len(chartParts)-2]
		subChart := false
		if len(chartParts) > 3 {
			subChart = chartParts[len(chartParts)-3] == "charts"
		}

		if !subChart {
			return true
		}

		if _, found := chartsList[AllSubCharts]; found {
			return true
		}

		if _, found := chartsList[chartName]; found {
			return true
		}

		return false
	}
	dirs := make([]string, 0)
	err := filepath.Walk(chartDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if chartFilter(path, info) {
			dirs = append(dirs, path)
		}

		return nil
	})
	return dirs, err
}

func parseChartCRDs(crds *[]unstructured.Unstructured, crdDir string) error {
	errOuter := filepath.Walk(crdDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error("Error parsing CustomResourceDefinition directory", "path", path, "error", err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Add to CRDs ..
		log.Debug("Parsing CustomResourceDefinition file", "path", path)
		b, err := os.ReadFile(path)
		if err != nil {
			log.Error("Failed to read CustomResourceDefinition file", "path", path, "error", err)
			return err
		}

		if len(b) == 0 {
			log.Warn("Empty CustomResourceDefinition file", "path", path)
			return nil
		}

		docs, err := parseCRDYamls(b)
		if err != nil {
			log.Error("Failed to parse YAML CustomResourceDefinition file", "path", path, "error", err)
			return err
		}
		dec := deser.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		for _, b := range docs {
			crd := unstructured.Unstructured{}

			_, gvk, err := dec.Decode(b, nil, &crd)
			if err != nil {
				log.Error("Failed to decode CustomResourceDefinition", "path", path, "error", err)
				continue
			}

			if gvk.Kind != "CustomResourceDefinition" {
				log.Error("File is not a CustomResourceDefinition", "path", path, "kind", gvk.Kind)
				continue
			}

			*crds = append(*crds, crd)
		}

		return err
	})

	return errOuter
}

func parseCRDYamls(b []byte) ([][]byte, error) {
	docs := [][]byte{}
	reader := k8syaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(b)))
	for {
		// Read document
		doc, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		docs = append(docs, doc)
	}

	return docs, nil
}
