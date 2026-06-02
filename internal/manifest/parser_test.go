package manifest

import (
	"testing"
)

func TestParse(t *testing.T) {
	yamlData := []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-api
  namespace: production
---
apiVersion: v1
kind: Service
metadata:
  name: payment-svc
# No namespace
---
# Empty doc
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
`)

	resources, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}

	if resources[0].Kind != "Deployment" || resources[0].Name != "payment-api" || resources[0].Namespace != "production" {
		t.Errorf("unexpected resource[0]: %+v", resources[0])
	}

	if resources[1].Kind != "Service" || resources[1].Name != "payment-svc" || resources[1].Namespace != "" {
		t.Errorf("unexpected resource[1]: %+v", resources[1])
	}

	if resources[2].Kind != "ConfigMap" || resources[2].Name != "my-config" || resources[2].Namespace != "" {
		t.Errorf("unexpected resource[2]: %+v", resources[2])
	}
}
