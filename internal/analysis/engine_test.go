package analysis_test

import (
	"context"
	"testing"

	"github.com/samaasi/kubectl-plan/internal/analysis"
	"github.com/samaasi/kubectl-plan/internal/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEngine_Analyze(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "payment-api", Namespace: "production"},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: "fake-cluster-uid"},
		},
	)

	client := &k8s.Client{
		Clientset: fakeClient,
		Namespace: "production",
	}

	engine := analysis.NewEngine(client, nil)
	res, err := engine.Analyze(context.Background(), "scale --replicas=0", "deployment", "payment-api")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Action != "scale --replicas=0" {
		t.Errorf("expected action 'scale --replicas=0', got %q", res.Action)
	}
	if res.Target.Name != "payment-api" {
		t.Errorf("expected target 'payment-api', got %q", res.Target.Name)
	}
	if res.ClusterUID != "fake-cluster-uid" {
		t.Errorf("expected cluster UID 'fake-cluster-uid', got %q", res.ClusterUID)
	}
}
