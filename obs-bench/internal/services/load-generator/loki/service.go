package loki_load_generator_service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"obs-bench/internal/pkg/portforwardhttp"
	load_generator_service "obs-bench/internal/services/load-generator"
)

var _ load_generator_service.IStackLoadGenerator = &service{}

// Три типа запросов имитируют типичную нагрузку Grafana-дашборда над Loki:
// простой поиск по потоку, rate-запрос и count_over_time.
// QPS=1 фиксирован как контрольная переменная — нагрузка на чтение постоянна
// во всех экспериментах, чтобы изолировать стоимость ingestion.
type querySpec struct {
	expr   string
	window string // длина окна для query_range
}

var queries = []querySpec{
	// простой log-stream — самый частый тип запроса в Grafana/Loki
	{expr: `{job="bench"}`, window: "5m"},
	// скорость появления строк за окно — метрический LogQL-запрос
	{expr: `rate({job="bench"}[5m])`, window: "15m"},
	// количество строк за более длинное окно — нагружает сканирование chunks
	{expr: `count_over_time({job="bench"}[15m])`, window: "30m"},
}

type service struct {
	counter atomic.Uint64
}

func NewLokiLoadGeneratorService() load_generator_service.IStackLoadGenerator {
	return &service{}
}

func (s *service) GenerateQueries(ctx context.Context, port int) error {
	idx := s.counter.Add(1) - 1
	q := queries[idx%uint64(len(queries))]

	end := time.Now().UnixNano()
	windowDur, _ := time.ParseDuration(q.window)
	start := time.Now().Add(-windowDur).UnixNano()

	u := fmt.Sprintf(
		"http://localhost:%d/loki/api/v1/query_range?query=%s&limit=50&start=%d&end=%d",
		port,
		url.QueryEscape(q.expr),
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
