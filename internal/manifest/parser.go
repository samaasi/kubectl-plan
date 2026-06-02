package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type Resource struct {
	Kind      string
	Name      string
	Namespace string
}

func ParseFile(filepath string) ([]Resource, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return Parse(data)
}

func Parse(data []byte) ([]Resource, error) {
	var resources []Resource
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))

	for {
		doc, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read YAML document: %w", err)
		}

		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		// Unmarshal into an unstructured map to extract standard metadata
		var obj map[string]interface{}
		if err := yaml.Unmarshal(doc, &obj); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}

		if obj == nil {
			continue
		}

		kind, _ := obj["kind"].(string)

		metadata, ok := obj["metadata"].(map[string]interface{})
		if !ok {
			// Skip things without metadata (e.g. lists or empty documents)
			continue
		}

		name, _ := metadata["name"].(string)
		namespace, _ := metadata["namespace"].(string)

		if kind != "" && name != "" {
			resources = append(resources, Resource{
				Kind:      kind,
				Name:      name,
				Namespace: namespace,
			})
		}
	}

	return resources, nil
}
