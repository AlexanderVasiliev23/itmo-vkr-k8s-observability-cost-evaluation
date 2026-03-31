package prometheus_load_generator_service

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

func NewPrometheusLoadGeneratorService() load_generator_service.IStackLoadGenerator {
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
		return fmt.Errorf("prometheus query: %w", err)
	}
	defer portforwardhttp.CloseResp(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}

	return nil
}
