package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	metric2 "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	slog.Info("starting metrics provider")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT env variable is required")
	}

	seriesCountStr := os.Getenv("SERIES_COUNT")
	if seriesCountStr == "" {
		log.Fatal("SERIES_COUNT env variable is required")
	}
	seriesCount, err := strconv.Atoi(seriesCountStr)
	if err != nil {
		log.Fatalf("invalid SERIES_COUNT: %v", err)
	}

	exporter, err := prometheusexporter.New()
	if err != nil {
		log.Fatal(err)
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	defer provider.Shutdown(context.Background())
	otel.SetMeterProvider(provider)

	meter := otel.Meter("metrics-provider")

	histograms := make([]metric2.Float64Histogram, seriesCount)
	for i := range seriesCount {
		seriesID := fmt.Sprintf("series_%04d", i)

		histograms[i], err = meter.Float64Histogram("request_duration_seconds",
			metric2.WithDescription("Simulated request duration distribution"),
			metric2.WithUnit("s"),
		)
		if err != nil {
			log.Fatal(err)
		}

		go generateSeriesMetrics(histograms[i], seriesID)
	}

	slog.Info("metrics provider started", "series_count", seriesCount)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func generateSeriesMetrics(histogram metric2.Float64Histogram, seriesID string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	attrs := metric2.WithAttributes(attribute.String("series_id", seriesID))

	for range ticker.C {
		histogram.Record(context.Background(), rand.Float64()*rand.Float64(), attrs)
	}
}
