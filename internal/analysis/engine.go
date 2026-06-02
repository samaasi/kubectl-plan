package analysis

import (
	"context"

	"github.com/samaasi/kubectl-plan/internal/criticality"
	"github.com/samaasi/kubectl-plan/internal/dependency"
	"github.com/samaasi/kubectl-plan/internal/k8s"
	"github.com/samaasi/kubectl-plan/internal/prometheus"
	"github.com/samaasi/kubectl-plan/internal/risk"
)

type Engine struct {
	client     *k8s.Client
	promClient *prometheus.Client
}

func NewEngine(client *k8s.Client, promClient *prometheus.Client) *Engine {
	return &Engine{client: client, promClient: promClient}
}

func (e *Engine) Analyze(ctx context.Context, action, targetKind, targetName string) (*AnalysisResult, error) {
	fetcher := k8s.NewFetcher(e.client)
	data, err := fetcher.FetchAll(ctx, e.client.Namespace)
	if err != nil {
		return nil, err
	}

	resolver := dependency.NewResolver(data, e.promClient)
	graph, err := resolver.Resolve(targetKind, targetName, e.client.Namespace)
	if err != nil {
		return nil, err
	}

	profile, _ := criticality.LoadProfile()

	scorer := risk.NewScorer(profile)
	riskScore, uncertainty := scorer.Score(graph)

	clusterUID, _ := e.client.GetClusterUID(ctx)

	hasCrossNamespace := false
	for _, node := range graph.Nodes {
		if node.Namespace != e.client.Namespace {
			hasCrossNamespace = true
			break
		}
	}

	overallConfidence := 0.65
	if len(graph.Edges) > 0 {
		maxConf := 0.0
		for _, edge := range graph.Edges {
			if edge.Confidence > maxConf {
				maxConf = edge.Confidence
			}
		}
		overallConfidence = maxConf
	}

	promAvailable := false
	if e.promClient != nil && e.promClient.IsReachable() {
		promAvailable = true
	}

	dataSources := DataSources{
		PrometheusAvailable: promAvailable,
		ServiceMeshDetected: false,
		K8sAPIAvailable:     true,
	}

	result := &AnalysisResult{
		Action: action,
		Target: graph.Target,
		Risk:   riskScore,
		Confidence: OverallConfidence{
			Overall: overallConfidence,
			Sources: []string{"Kubernetes topology"},
		},
		Uncertainty:    uncertainty,
		Graph:          *graph,
		CrossNamespace: hasCrossNamespace,
		ClusterUID:     clusterUID,
		ClusterVersion: ClusterVersion{Major: "1", Minor: "29"},
		DataSources:    dataSources,
	}

	return result, nil
}
