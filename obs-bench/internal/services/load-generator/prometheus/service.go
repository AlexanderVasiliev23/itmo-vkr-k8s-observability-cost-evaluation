package prometheus_load_generator_service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"obs-bench/internal/pkg/portforwardhttp"
	load_generator_service "obs-bench/internal/services/load-generator"
)

var _ load_generator_service.IStackLoadGenerator = &service{}

// Три типа запросов имитируют типичную нагрузку Grafana-дашборда: простой counter,
// агрегацию и гистограммный перцентиль. QPS=1 фиксирован как контрольная переменная —
// нагрузка на чтение постоянна во всех экспериментах, чтобы изолировать стоимость ingestion.
var queries = []string{
	// мгновенная скорость counter-метрики — самый частый тип запроса в дашбордах
	"rate(request_duration_seconds_count[1m])",
	// агрегация по всем сериям — нагружает движок сортировкой и слиянием
	"sum by (method) (rate(request_duration_seconds_count[1m]))",
	// гистограммный перцентиль — дорогой вычислительно запрос
	"histogram_quantile(0.99, sum by (le) (rate(request_duration_seconds_bucket[5m])))",
}

type service struct {
	counter atomic.Uint64
}

func NewPrometheusLoadGeneratorService() load_generator_service.IStackLoadGenerator {
	return &service{}
}

func (s *service) GenerateQueries(ctx context.Context, port int) error {
	idx := s.counter.Add(1) - 1
	query := queries[idx%uint64(len(queries))]

	queryURL := fmt.Sprintf("http://localhost:%d/api/v1/query", port)
	params := url.Values{}
	params.Set("query", query)

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
