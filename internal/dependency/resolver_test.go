package dependency_test

import (
	"testing"

	"github.com/samaasi/kubectl-plan/internal/dependency"
	"github.com/samaasi/kubectl-plan/internal/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeDeployment(name, ns string, labels map[string]string) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
		},
	}
}

func makeService(name, ns string, selector map[string]string) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       corev1.ServiceSpec{Selector: selector},
	}
}

func makePod(name, ns string, ownerKind, ownerName string, labels map[string]string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: ownerKind, Name: ownerName},
			},
		},
	}
}

func makeReplicaSet(name, ns, deployName string) appsv1.ReplicaSet {
	return appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "Deployment", Name: deployName},
			},
		},
	}
}

// TestResolve_deploymentNotFound verifies an unfound deployment still returns a graph.
func TestResolve_deploymentNotFound(t *testing.T) {
	data := &k8s.ClusterData{}
	r := dependency.NewResolver(data, nil)
	graph, err := r.Resolve("deployment", "missing", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if graph.Target.Name != "missing" {
		t.Errorf("expected target name 'missing', got %q", graph.Target.Name)
	}
}

// TestResolve_serviceSelection verifies service label selector matching.
func TestResolve_serviceSelection(t *testing.T) {
	labels := map[string]string{"app": "payment-api"}
	rs := makeReplicaSet("payment-api-rs", "production", "payment-api")
	pod := makePod("payment-api-pod-1", "production", "ReplicaSet", "payment-api-rs", labels)
	svc := makeService("payment-svc", "production", labels)
	deploy := makeDeployment("payment-api", "production", labels)

	data := &k8s.ClusterData{
		Deployments: []appsv1.Deployment{deploy},
		ReplicaSets: []appsv1.ReplicaSet{rs},
		Pods:        []corev1.Pod{pod},
		Services:    []corev1.Service{svc},
	}

	r := dependency.NewResolver(data, nil)
	graph, err := r.Resolve("deployment", "payment-api", "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, e := range graph.Edges {
		if e.Relationship == dependency.RelSelects {
			found = true
			if e.Confidence != 0.95 {
				t.Errorf("expected confidence 0.95 for SELECTS edge, got %v", e.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected at least one SELECTS edge from service to deployment")
	}
}

// TestResolve_envVarReference verifies cross-service env var dependency detection.
func TestResolve_envVarReference(t *testing.T) {
	labels := map[string]string{"app": "payment-api"}
	rs := makeReplicaSet("payment-api-rs", "production", "payment-api")
	pod := makePod("payment-api-pod", "production", "ReplicaSet", "payment-api-rs", labels)
	svc := makeService("payment-svc", "production", labels)
	target := makeDeployment("payment-api", "production", labels)

	consumerLabels := map[string]string{"app": "checkout"}
	consumer := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "production", Labels: consumerLabels},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: consumerLabels},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "checkout",
							Env: []corev1.EnvVar{
								{Name: "PAYMENT_URL", Value: "http://payment-svc/api"},
							},
						},
					},
				},
			},
		},
	}

	data := &k8s.ClusterData{
		Deployments: []appsv1.Deployment{target, consumer},
		ReplicaSets: []appsv1.ReplicaSet{rs},
		Pods:        []corev1.Pod{pod},
		Services:    []corev1.Service{svc},
	}

	r := dependency.NewResolver(data, nil)
	graph, err := r.Resolve("deployment", "payment-api", "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, e := range graph.Edges {
		if e.Relationship == dependency.RelEnvRef {
			found = true
			if e.Confidence > 0.75 {
				t.Errorf("env var evidence should have confidence <= 0.75, got %v", e.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected ENV_REF edge from consumer referencing payment-svc")
	}
}

// TestResolve_statefulSet verifies StatefulSet resolution.
func TestResolve_statefulSet(t *testing.T) {
	labels := map[string]string{"app": "postgres"}
	pod := makePod("postgres-0", "data", "StatefulSet", "postgres", labels)
	sts := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "postgres", Namespace: "data", Labels: labels},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
		},
	}

	data := &k8s.ClusterData{
		StatefulSets: []appsv1.StatefulSet{sts},
		Pods:         []corev1.Pod{pod},
	}

	r := dependency.NewResolver(data, nil)
	graph, err := r.Resolve("statefulset", "postgres", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if graph.Target.Kind != "StatefulSet" {
		t.Errorf("expected target kind StatefulSet, got %q", graph.Target.Kind)
	}

	ownsFound := false
	for _, e := range graph.Edges {
		if e.Relationship == dependency.RelOwns {
			ownsFound = true
		}
	}
	if !ownsFound {
		t.Error("expected OWNS edge from StatefulSet to pod")
	}
}
