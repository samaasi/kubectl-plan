package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/samaasi/kubectl-plan/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	BaseURL  string
	Lookback string

	// For proxying through k8s API
	k8sClient *k8s.Client
	svcName   string
	svcNS     string
	svcPort   string
}

func NewClient(baseURL, lookback string) *Client {
	if lookback == "" {
		lookback = "24h"
	}
	return &Client{
		BaseURL:  baseURL,
		Lookback: lookback,
	}
}

func (c *Client) IsReachable() bool {
	return c.BaseURL != "" || (c.k8sClient != nil && c.svcName != "")
}

func (c *Client) Discover(ctx context.Context, k8sClient *k8s.Client) (string, error) {
	c.k8sClient = k8sClient

	// Search across all namespaces
	services, err := k8sClient.Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, svc := range services.Items {
		nameMatch := strings.Contains(svc.Name, "prometheus") && !strings.Contains(svc.Name, "alertmanager")
		labelMatch := false
		for k, v := range svc.Labels {
			if strings.Contains(strings.ToLower(k), "app") && strings.Contains(strings.ToLower(v), "prometheus") {
				labelMatch = true
			}
		}

		if nameMatch || labelMatch {
			port := "9090"
			for _, p := range svc.Spec.Ports {
				if p.Port == 9090 || p.Name == "http" || p.Name == "web" {
					port = fmt.Sprintf("%d", p.Port)
					break
				}
			}
			c.svcName = svc.Name
			c.svcNS = svc.Namespace
			c.svcPort = port
			return fmt.Sprintf("k8s-proxy://%s/%s:%s", c.svcNS, c.svcName, c.svcPort), nil
		}
	}

	return "", fmt.Errorf("prometheus service not found")
}

func (c *Client) Query(ctx context.Context, query string) (float64, error) {
	if !c.IsReachable() {
		return 0, fmt.Errorf("prometheus client not reachable")
	}

	params := url.Values{}
	params.Add("query", query)

	var body []byte
	var err error

	if c.BaseURL != "" && !strings.HasPrefix(c.BaseURL, "k8s-proxy://") {
		u := fmt.Sprintf("%s/api/v1/query?%s", strings.TrimRight(c.BaseURL, "/"), params.Encode())
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, errReq := http.DefaultClient.Do(req)
		if errReq != nil {
			return 0, errReq
		}
		defer resp.Body.Close()
		body = make([]byte, 1<<16)
		n, _ := resp.Body.Read(body)
		body = body[:n]
	} else {
		req := c.k8sClient.Clientset.CoreV1().Services(c.svcNS).ProxyGet("http", c.svcName, c.svcPort, "/api/v1/query", map[string]string{
			"query": query,
		})
		body, err = req.DoRaw(ctx)
		if err != nil {
			return 0, err
		}
	}

	var res struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return 0, err
	}

	if res.Status != "success" || len(res.Data.Result) == 0 {
		return 0, nil
	}

	valArr := res.Data.Result[0].Value
	if len(valArr) > 1 {
		if valStr, ok := valArr[1].(string); ok {
			var rate float64
			fmt.Sscanf(valStr, "%f", &rate)
			return rate, nil
		}
	}

	return 0, nil
}

func (c *Client) GetTrafficRate(ctx context.Context, sourceSvc, destSvc string) (float64, error) {
	// A generic query for service-to-service traffic
	// Usually metrics look like: istio_requests_total{source_app="...", destination_service="..."}
	// Or similar. We will just use a generic http_requests_total query.
	query := fmt.Sprintf("sum(rate(http_requests_total{destination_service=~\"%s.*\"}[%s]))", destSvc, c.Lookback)
	rate, err := c.Query(ctx, query)
	if err != nil {
		return 0, err
	}
	if rate == 0 {
		// Try a simpler query just checking if destination was hit at all in case source labels are missing
		query = fmt.Sprintf("sum(rate(http_requests_total{job=~\"%s.*\"}[%s]))", destSvc, c.Lookback)
		return c.Query(ctx, query)
	}
	return rate, nil
}
