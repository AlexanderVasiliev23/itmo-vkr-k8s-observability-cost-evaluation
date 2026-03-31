package loki_load_generator_service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"obs-bench/internal/pkg/portforwardhttp"
	load_generator_service "obs-bench/internal/services/load-generator"
)

var _ load_generator_service.IStackLoadGenerator = &service{}

type service struct{}

func NewLokiLoadGeneratorService() load_generator_service.IStackLoadGenerator {
	return &service{}
}

func (s *service) GenerateQueries(ctx context.Context, port int) error {
	end := time.Now().UnixNano()
	start := time.Now().Add(-5 * time.Minute).UnixNano()
	q := `{job="bench"}`
	u := fmt.Sprintf(
		"http://localhost:%d/loki/api/v1/query_range?query=%s&limit=50&start=%d&end=%d",
		port,
		url.QueryEscape(q),
		start,
		end,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("build loki query: %w", err)
	}
	req.Close = true
	resp, err := portforwardhttp.Client.Do(req)
	if err != nil {
		return fmt.Errorf("loki query_range: %w", err)
	}
	defer portforwardhttp.CloseResp(resp)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("loki returned status %d", resp.StatusCode)
	}
	return nil
}
