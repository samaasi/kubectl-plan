package prometheus

import (
	"context"
	"testing"

	"github.com/samaasi/kubectl-plan/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDiscover(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-server",
			Namespace: "monitoring",
			Labels:    map[string]string{"app": "prometheus"},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: 9090, Name: "http"},
			},
		},
	})

	k8sC := &k8s.Client{Clientset: fakeClient}
	promClient := NewClient("", "24h")

	url, err := promClient.Discover(context.Background(), k8sC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "k8s-proxy://monitoring/prometheus-server:9090"
	if url != expected {
		t.Errorf("expected %q, got %q", expected, url)
	}

	if !promClient.IsReachable() {
		t.Error("expected client to be reachable after discovery")
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:9090", "")
	if c.Lookback != "24h" {
		t.Errorf("expected default lookback 24h, got %s", c.Lookback)
	}
	if c.BaseURL != "http://localhost:9090" {
		t.Errorf("expected BaseURL http://localhost:9090, got %s", c.BaseURL)
	}
	if !c.IsReachable() {
		t.Error("expected client with BaseURL to be reachable")
	}
}
