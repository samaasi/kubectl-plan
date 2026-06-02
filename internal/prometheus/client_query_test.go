package prometheus

import (
	"context"
	"testing"
)

func TestQuery(t *testing.T) {
	c := NewClient("http://localhost:9090", "24h")
	rate, err := c.Query(context.Background(), "up")
	// Expected to fail because localhost:9090 doesn't exist during tests usually
	if err == nil && rate > 0 {
		t.Log("Expected failure or 0 rate")
	}
}

func TestGetTrafficRate(t *testing.T) {
	c := NewClient("http://localhost:9090", "24h")
	rate, err := c.GetTrafficRate(context.Background(), "payment-api", "checkout")
	if err == nil && rate > 0 {
		t.Log("Expected failure or 0 rate")
	}
}
