package helmutil

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	deser "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func FilterCharts(unstructs *[]unstructured.Unstructured, dir string, types []string) error {
	errOuter := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error("Error parsing CustomResourceDefinition directory", "path", path, "error", err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Add to CRDs ..
		log.Debug("Parsing YAML file", "path", path)
		b, err := os.ReadFile(path)
		if err != nil {
			log.Error("Failed to read YAML file", "path", path, "error", err)
			return err
		}

		if len(b) == 0 {
			log.Warn("Empty YAML file", "path", path)
			return nil
		}

		docs, err := parseYamlDocs(b)
		if err != nil {
			log.Error("Failed to parse YAML file", "path", path, "error", err)
			return err
		}
		dec := deser.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

		for _, b := range docs {
			crd := unstructured.Unstructured{}

			_, gvk, err := dec.Decode(b, nil, &crd)
			if err != nil {
				log.Error("Failed to decode YAML to Unstructured", "path", path, "error", err)
				continue
			}

			if !slices.Contains(types, gvk.Kind) {
				continue
			}

			*unstructs = append(*unstructs, crd)
		}

		return err
	})

	return errOuter
}

func parseYamlDocs(b []byte) ([][]byte, error) {
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
