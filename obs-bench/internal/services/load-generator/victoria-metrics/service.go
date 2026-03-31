package victoria_metrics_load_generator_service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"obs-bench/internal/pkg/portforwardhttp"
	load_generator_service "obs-bench/internal/services/load-generator"
)

var _ load_generator_service.IStackLoadGenerator = &service{}

type service struct {
}

// NewVictoriaMetricsLoadGeneratorService создаёт генератор нагрузки к VictoriaMetrics single-node.
// Query API совместим с Prometheus (/api/v1/query, PromQL).
func NewVictoriaMetricsLoadGeneratorService() load_generator_service.IStackLoadGenerator {
	return &service{}
}

func (s *service) GenerateQueries(ctx context.Context, port int) error {
	queryURL := fmt.Sprintf("http://localhost:%d/api/v1/query", port)
	params := url.Values{}
	params.Set("query", "rate(request_duration_seconds_count[1m])")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL+"?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Close = true

	resp, err := portforwardhttp.Client.Do(req)
	if err != nil {
		return fmt.Errorf("victoria metrics query: %w", err)
	}
	defer portforwardhttp.CloseResp(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("victoria metrics returned status %d", resp.StatusCode)
	}

	return nil
}
