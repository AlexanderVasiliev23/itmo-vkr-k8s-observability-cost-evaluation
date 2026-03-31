package opensearch_load_generator_service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"obs-bench/internal/pkg/portforwardhttp"
	load_generator_service "obs-bench/internal/services/load-generator"
)

var _ load_generator_service.IStackLoadGenerator = &service{}

type service struct{}

func NewOpenSearchLoadGeneratorService() load_generator_service.IStackLoadGenerator {
	return &service{}
}

func (s *service) GenerateQueries(ctx context.Context, port int) error {
	body := strings.NewReader(`{"size":10,"query":{"match_all":{}}}`)
	u := fmt.Sprintf("http://localhost:%d/logbench/_search", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return fmt.Errorf("build opensearch search: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	resp, err := portforwardhttp.Client.Do(req)
	if err != nil {
		return fmt.Errorf("opensearch search: %w", err)
	}
	defer portforwardhttp.CloseResp(resp)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opensearch returned status %d", resp.StatusCode)
	}
	return nil
}
