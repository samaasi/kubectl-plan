package dependency

import (
	"context"
	"fmt"
	"strings"

	"github.com/samaasi/kubectl-plan/internal/k8s"
	"github.com/samaasi/kubectl-plan/internal/prometheus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type Resolver struct {
	data       *k8s.ClusterData
	promClient *prometheus.Client
}

func NewResolver(data *k8s.ClusterData, promClient *prometheus.Client) *Resolver {
	return &Resolver{data: data, promClient: promClient}
}

func (r *Resolver) Resolve(targetKind, targetName, targetNamespace string) (*DependencyGraph, error) {
	targetNode := r.findTargetNode(targetKind, targetName, targetNamespace)
	graph := NewDependencyGraph(targetNode)
	graph.AddNode(targetNode)

	var targetPods []corev1.Pod
	var targetSelectors []map[string]string

	switch strings.ToLower(targetKind) {
	case "deployment", "deployments", "deploy":
		r.resolveDeployment(graph, targetName, targetNamespace, &targetPods, &targetSelectors)
	case "statefulset", "statefulsets", "sts":
		r.resolveStatefulSet(graph, targetName, targetNamespace, &targetPods, &targetSelectors)
	case "daemonset", "daemonsets", "ds":
		r.resolveDaemonSet(graph, targetName, targetNamespace, &targetPods, &targetSelectors)
	}

	matchedServices := r.resolveServices(graph, targetNamespace, targetPods, targetSelectors)
	r.resolveIngresses(graph, targetNamespace, matchedServices)
	r.resolveWorkloadsEnv(graph, targetNamespace, matchedServices)
	r.resolveNetworkPolicies(graph, targetNamespace, targetPods)
	r.resolveConfigMaps(graph, targetNamespace, matchedServices)
	r.resolvePDBs(graph, targetNamespace, targetPods)
	r.resolveHPAs(graph, targetNamespace, targetNode)

	r.enrichWithTrafficData(graph)

	return graph, nil
}

func (r *Resolver) enrichWithTrafficData(graph *DependencyGraph) {
	if r.promClient == nil || !r.promClient.IsReachable() {
		return
	}
	
	// We only want to enrich edges that flow into the target (RelSelects, RelRoutes, RelEnvRef, RelVolumeRef)
	// Actually, traffic usually flows FROM the dependent TO the target.
	for i, edge := range graph.Edges {
		if edge.Relationship == RelOwns {
			continue // structural, no traffic needed
		}

		// Example: from dependent node to target
		fromParts := strings.Split(edge.From, "/")
		
		rate, err := r.promClient.GetTrafficRate(context.Background(), fromParts[1], graph.Target.Name)
		if err == nil && rate > 0 {
			graph.Edges[i].Confidence = 0.99
			graph.Edges[i].Evidence = append(graph.Edges[i].Evidence, Evidence{
				Type:        EvidencePrometheus,
				Source:      SourcePrometheus,
				Description: fmt.Sprintf("%.2f req/sec · destination_service=%s · Prometheus", rate, graph.Target.Name),
				Confidence:  0.99,
			})
		}
	}
}

func (r *Resolver) findTargetNode(targetKind, targetName, targetNamespace string) Node {
	switch strings.ToLower(targetKind) {
	case "deployment", "deployments", "deploy":
		for _, d := range r.data.Deployments {
			if d.Name == targetName && d.Namespace == targetNamespace {
				return Node{Kind: "Deployment", Name: d.Name, Namespace: d.Namespace, Labels: d.Labels, Metadata: map[string]string{}}
			}
		}
	case "statefulset", "statefulsets", "sts":
		for _, s := range r.data.StatefulSets {
			if s.Name == targetName && s.Namespace == targetNamespace {
				return Node{Kind: "StatefulSet", Name: s.Name, Namespace: s.Namespace, Labels: s.Labels, Metadata: map[string]string{}}
			}
		}
	case "daemonset", "daemonsets", "ds":
		for _, d := range r.data.DaemonSets {
			if d.Name == targetName && d.Namespace == targetNamespace {
				return Node{Kind: "DaemonSet", Name: d.Name, Namespace: d.Namespace, Labels: d.Labels, Metadata: map[string]string{}}
			}
		}
	}
	return Node{Kind: targetKind, Name: targetName, Namespace: targetNamespace, Labels: map[string]string{}, Metadata: map[string]string{}}
}

func (r *Resolver) resolveDeployment(graph *DependencyGraph, targetName, targetNamespace string, targetPods *[]corev1.Pod, targetSelectors *[]map[string]string) {
	var targetRS []appsv1.ReplicaSet
	for _, rs := range r.data.ReplicaSets {
		for _, ref := range rs.OwnerReferences {
			if ref.Kind == "Deployment" && ref.Name == targetName {
				targetRS = append(targetRS, rs)
			}
		}
	}
	for _, pod := range r.data.Pods {
		for _, rs := range targetRS {
			for _, ref := range pod.OwnerReferences {
				if ref.Kind == "ReplicaSet" && ref.Name == rs.Name {
					*targetPods = append(*targetPods, pod)
					podNode := Node{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace, Labels: pod.Labels}
					graph.AddNode(podNode)
					graph.AddEdge(Edge{
						From:         NodeKey(graph.Target.Namespace, graph.Target.Kind, graph.Target.Name),
						To:           NodeKey(podNode.Namespace, podNode.Kind, podNode.Name),
						Relationship: RelOwns, Depth: 1, Confidence: 1.0,
						Evidence: []Evidence{{Type: EvidenceOwnerRef, Source: SourceK8sAPI, Description: fmt.Sprintf("ownerReference via ReplicaSet %s", rs.Name), Confidence: 1.0}},
					})
				}
			}
		}
	}
	for _, d := range r.data.Deployments {
		if d.Name == targetName && d.Namespace == targetNamespace && d.Spec.Selector != nil {
			*targetSelectors = append(*targetSelectors, d.Spec.Selector.MatchLabels)
		}
	}
}

func (r *Resolver) resolveStatefulSet(graph *DependencyGraph, targetName, targetNamespace string, targetPods *[]corev1.Pod, targetSelectors *[]map[string]string) {
	for _, pod := range r.data.Pods {
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == "StatefulSet" && ref.Name == targetName {
				*targetPods = append(*targetPods, pod)
				podNode := Node{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace, Labels: pod.Labels}
				graph.AddNode(podNode)
				graph.AddEdge(Edge{
					From:         NodeKey(graph.Target.Namespace, graph.Target.Kind, graph.Target.Name),
					To:           NodeKey(podNode.Namespace, podNode.Kind, podNode.Name),
					Relationship: RelOwns, Depth: 1, Confidence: 1.0,
					Evidence: []Evidence{{Type: EvidenceOwnerRef, Source: SourceK8sAPI, Description: "direct ownerReference", Confidence: 1.0}},
				})
			}
		}
	}
	for _, s := range r.data.StatefulSets {
		if s.Name == targetName && s.Namespace == targetNamespace && s.Spec.Selector != nil {
			*targetSelectors = append(*targetSelectors, s.Spec.Selector.MatchLabels)
		}
	}
}

func (r *Resolver) resolveDaemonSet(graph *DependencyGraph, targetName, targetNamespace string, targetPods *[]corev1.Pod, targetSelectors *[]map[string]string) {
	for _, pod := range r.data.Pods {
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == "DaemonSet" && ref.Name == targetName {
				*targetPods = append(*targetPods, pod)
				podNode := Node{Kind: "Pod", Name: pod.Name, Namespace: pod.Namespace, Labels: pod.Labels}
				graph.AddNode(podNode)
				graph.AddEdge(Edge{
					From:         NodeKey(graph.Target.Namespace, graph.Target.Kind, graph.Target.Name),
					To:           NodeKey(podNode.Namespace, podNode.Kind, podNode.Name),
					Relationship: RelOwns, Depth: 1, Confidence: 1.0,
					Evidence: []Evidence{{Type: EvidenceOwnerRef, Source: SourceK8sAPI, Description: "direct ownerReference", Confidence: 1.0}},
				})
			}
		}
	}
	for _, ds := range r.data.DaemonSets {
		if ds.Name == targetName && ds.Namespace == targetNamespace && ds.Spec.Selector != nil {
			*targetSelectors = append(*targetSelectors, ds.Spec.Selector.MatchLabels)
		}
	}
}

func (r *Resolver) resolveServices(graph *DependencyGraph, targetNamespace string, targetPods []corev1.Pod, targetSelectors []map[string]string) []corev1.Service {
	var matchedServices []corev1.Service
	for _, svc := range r.data.Services {
		if svc.Namespace != targetNamespace {
			continue
		}
		matches := false
		if len(svc.Spec.Selector) > 0 {
			for _, pod := range targetPods {
				if labelSelectorContains(svc.Spec.Selector, pod.Labels) {
					matches = true
					break
				}
			}
			if !matches {
				for _, sel := range targetSelectors {
					if labelSelectorSubset(svc.Spec.Selector, sel) {
						matches = true
						break
					}
				}
			}
		}
		if matches {
			matchedServices = append(matchedServices, svc)
			svcNode := Node{Kind: "Service", Name: svc.Name, Namespace: svc.Namespace, Labels: svc.Labels}
			graph.AddNode(svcNode)
			graph.AddEdge(Edge{
				From:         NodeKey(svcNode.Namespace, svcNode.Kind, svcNode.Name),
				To:           NodeKey(graph.Target.Namespace, graph.Target.Kind, graph.Target.Name),
				Relationship: RelSelects, Depth: 1, Confidence: 0.95,
				Evidence: []Evidence{{Type: EvidenceLabelSelector, Source: SourceK8sAPI, Description: fmt.Sprintf("selector %v matches target pods/labels", svc.Spec.Selector), Confidence: 0.95}},
			})
		}
	}
	return matchedServices
}

func (r *Resolver) resolveIngresses(graph *DependencyGraph, targetNamespace string, matchedServices []corev1.Service) {
	for _, ingress := range r.data.Ingresses {
		if ingress.Namespace != targetNamespace {
			continue
		}
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service == nil {
					continue
				}
				for _, svc := range matchedServices {
					if path.Backend.Service.Name == svc.Name {
						ingNode := Node{Kind: "Ingress", Name: ingress.Name, Namespace: ingress.Namespace, Labels: ingress.Labels}
						graph.AddNode(ingNode)
						graph.AddEdge(Edge{
							From:         NodeKey(ingNode.Namespace, ingNode.Kind, ingNode.Name),
							To:           NodeKey(svc.Namespace, svc.Kind, svc.Name),
							Relationship: RelRoutes, Depth: 2, Confidence: 0.95,
							Evidence: []Evidence{{Type: EvidenceIngressBackend, Source: SourceK8sAPI, Description: fmt.Sprintf("rules route %s -> service %s", path.Path, svc.Name), Confidence: 0.95}},
						})
					}
				}
			}
		}
	}
}

func (r *Resolver) resolveWorkloadsEnv(graph *DependencyGraph, targetNamespace string, matchedServices []corev1.Service) {
	for _, d := range r.data.Deployments {
		if d.Name == graph.Target.Name && d.Namespace == targetNamespace {
			continue
		}
		r.scanWorkloadEnv(graph, graph.Target, matchedServices, d.Name, d.Namespace, "Deployment", d.Spec.Template.Spec.Containers, d.Labels)
	}
	for _, s := range r.data.StatefulSets {
		if s.Name == graph.Target.Name && s.Namespace == targetNamespace {
			continue
		}
		r.scanWorkloadEnv(graph, graph.Target, matchedServices, s.Name, s.Namespace, "StatefulSet", s.Spec.Template.Spec.Containers, s.Labels)
	}
	for _, d := range r.data.DaemonSets {
		if d.Name == graph.Target.Name && d.Namespace == targetNamespace {
			continue
		}
		r.scanWorkloadEnv(graph, graph.Target, matchedServices, d.Name, d.Namespace, "DaemonSet", d.Spec.Template.Spec.Containers, d.Labels)
	}
}

func (r *Resolver) resolveNetworkPolicies(graph *DependencyGraph, targetNamespace string, targetPods []corev1.Pod) {
	for _, policy := range r.data.NetPolicies {
		if policy.Namespace != targetNamespace {
			continue
		}
		for _, ingressRule := range policy.Spec.Ingress {
			for _, from := range ingressRule.From {
				if from.PodSelector != nil {
					matches := false
					for _, pod := range targetPods {
						if labelSelectorContains(from.PodSelector.MatchLabels, pod.Labels) {
							matches = true
							break
						}
					}
					if matches {
						policyNode := Node{Kind: "NetworkPolicy", Name: policy.Name, Namespace: policy.Namespace, Labels: policy.Labels}
						graph.AddNode(policyNode)
						graph.AddEdge(Edge{
							From:         NodeKey(policyNode.Namespace, policyNode.Kind, policyNode.Name),
							To:           NodeKey(graph.Target.Namespace, graph.Target.Kind, graph.Target.Name),
							Relationship: RelNetworkPolicy, Depth: 1, Confidence: 0.80,
							Evidence: []Evidence{{Type: EvidenceNetworkPolicy, Source: SourceK8sAPI, Description: fmt.Sprintf("ingress policy %s restricts access to target", policy.Name), Confidence: 0.80}},
						})
					}
				}
			}
		}
	}
}

func (r *Resolver) resolveConfigMaps(graph *DependencyGraph, targetNamespace string, matchedServices []corev1.Service) {
	for _, cm := range r.data.ConfigMaps {
		if cm.Namespace != targetNamespace {
			continue
		}
		for _, svc := range matchedServices {
			cmMatched := false
			for key, val := range cm.Data {
				if strings.Contains(val, svc.Name) {
					cmMatched = true
					cmNode := Node{Kind: "ConfigMap", Name: cm.Name, Namespace: cm.Namespace, Labels: cm.Labels}
					graph.AddNode(cmNode)
					graph.AddEdge(Edge{
						From:         NodeKey(cmNode.Namespace, cmNode.Kind, cmNode.Name),
						To:           NodeKey(svc.Namespace, svc.Kind, svc.Name),
						Relationship: RelVolumeRef, Depth: 2, Confidence: 0.60,
						Evidence: []Evidence{{Type: EvidenceVolumeMount, Source: SourceK8sAPI, Description: fmt.Sprintf("ConfigMap key %s references service %s", key, svc.Name), Confidence: 0.60, RawValue: val}},
					})
					break
				}
			}
			if cmMatched {
				break
			}
		}
	}
}

func (r *Resolver) resolvePDBs(graph *DependencyGraph, targetNamespace string, targetPods []corev1.Pod) {
	for _, pdb := range r.data.PDBs {
		if pdb.Namespace != targetNamespace {
			continue
		}
		matches := false
		if pdb.Spec.Selector != nil {
			for _, pod := range targetPods {
				if labelSelectorContains(pdb.Spec.Selector.MatchLabels, pod.Labels) {
					matches = true
					break
				}
			}
		}
		if matches {
			pdbNode := Node{
				Kind: "PodDisruptionBudget", Name: pdb.Name, Namespace: pdb.Namespace, Labels: pdb.Labels,
				Metadata: map[string]string{"minAvailable": fmt.Sprintf("%v", pdb.Spec.MinAvailable)},
			}
			graph.AddNode(pdbNode)
			graph.AddEdge(Edge{
				From:         NodeKey(pdbNode.Namespace, pdbNode.Kind, pdbNode.Name),
				To:           NodeKey(graph.Target.Namespace, graph.Target.Kind, graph.Target.Name),
				Relationship: RelOwns, Depth: 1, Confidence: 0.95,
				Evidence: []Evidence{{Type: EvidenceLabelSelector, Source: SourceK8sAPI, Description: fmt.Sprintf("PDB matches target pod labels (minAvailable=%v)", pdb.Spec.MinAvailable), Confidence: 0.95}},
			})
		}
	}
}

func (r *Resolver) resolveHPAs(graph *DependencyGraph, targetNamespace string, targetNode Node) {
	for _, hpa := range r.data.HPAs {
		if hpa.Namespace != targetNamespace {
			continue
		}
		if hpa.Spec.ScaleTargetRef.Kind == targetNode.Kind && hpa.Spec.ScaleTargetRef.Name == targetNode.Name {
			hpaNode := Node{Kind: "HorizontalPodAutoscaler", Name: hpa.Name, Namespace: hpa.Namespace, Labels: hpa.Labels}
			graph.AddNode(hpaNode)
			graph.AddEdge(Edge{
				From:         NodeKey(hpaNode.Namespace, hpaNode.Kind, hpaNode.Name),
				To:           NodeKey(targetNode.Namespace, targetNode.Kind, targetNode.Name),
				Relationship: RelOwns, Depth: 1, Confidence: 0.95,
				Evidence: []Evidence{{Type: EvidenceLabelSelector, Source: SourceK8sAPI, Description: fmt.Sprintf("HPA targets %s/%s directly", targetNode.Kind, targetNode.Name), Confidence: 0.95}},
			})
		}
	}
}

func (r *Resolver) scanWorkloadEnv(
	graph *DependencyGraph,
	targetNode Node,
	matchedServices []corev1.Service,
	name, namespace, kind string,
	containers []corev1.Container,
	labels map[string]string,
) {
	for _, container := range containers {
		for _, env := range container.Env {
			for _, svc := range matchedServices {
				dnsPattern1 := svc.Name
				dnsPattern2 := fmt.Sprintf("%s.%s", svc.Name, svc.Namespace)
				dnsPattern3 := fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace)

				if env.Value == dnsPattern1 || env.Value == dnsPattern2 || env.Value == dnsPattern3 ||
					strings.Contains(env.Value, dnsPattern1) || strings.Contains(env.Value, dnsPattern2) {
					wNode := Node{
						Kind:      kind,
						Name:      name,
						Namespace: namespace,
						Labels:    labels,
					}
					graph.AddNode(wNode)

					evidenceType := EvidenceEnvVar
					confidence := 0.70
					description := fmt.Sprintf("env.%s references service %s", env.Name, svc.Name)

					if strings.Contains(env.Value, ".cluster.local") || strings.Contains(env.Value, fmt.Sprintf("%s.", svc.Name)) {
						evidenceType = EvidenceDNSPattern
						confidence = 0.65
						description = fmt.Sprintf("env.%s contains cluster DNS pattern for %s", env.Name, svc.Name)
					}

					graph.AddEdge(Edge{
						From:         NodeKey(wNode.Namespace, wNode.Kind, wNode.Name),
						To:           NodeKey(svc.Namespace, svc.Kind, svc.Name),
						Relationship: RelEnvRef,
						Depth:        2,
						Confidence:   confidence,
						Evidence: []Evidence{
							{
								Type:        evidenceType,
								Source:      SourceK8sAPI,
								Description: description,
								Confidence:  confidence,
								RawValue:    env.Value,
							},
						},
					})
				}
			}
		}
	}
}

func labelSelectorContains(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func labelSelectorSubset(selector, targetSelector map[string]string) bool {
	if len(selector) == 0 || len(targetSelector) == 0 {
		return false
	}
	for k, v := range selector {
		if targetSelector[k] != v {
			return false
		}
	}
	return true
}
