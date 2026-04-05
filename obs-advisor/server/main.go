package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"obs-advisor-server/capacitymodel"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "", "путь до SQLite-базы obs-bench (обязательный)")
	port := flag.String("port", "8080", "порт HTTP-сервера")
	staticDir := flag.String("static", ".", "директория со статическими файлами (index.html и т.п.)")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "error: --db обязательный флаг")
		os.Exit(1)
	}

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("ping db", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	// GET /api/measurements — все точки из SQLite.
	// Формат совместим с data.json: JSON-массив объектов с полями snake_case.
	mux.HandleFunc("/api/measurements", func(w http.ResponseWriter, r *http.Request) {
		rows, err := loadRows(db)
		if err != nil {
			slog.Error("load rows", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	})

	// GET /api/estimate — оценка ресурсов по модели.
	// Параметры: instrument, workload_type, target_load, target_retention_days, error_budget (опц., default 0.2).
	mux.HandleFunc("/api/estimate", func(w http.ResponseWriter, r *http.Request) {
		rows, err := loadRows(db)
		if err != nil {
			slog.Error("load rows", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		q := r.URL.Query()
		targetLoad, _ := strconv.ParseFloat(q.Get("target_load"), 64)
		targetRetention, _ := strconv.ParseFloat(q.Get("target_retention_days"), 64)
		errorBudget, _ := strconv.ParseFloat(q.Get("error_budget"), 64)
		if errorBudget == 0 {
			errorBudget = capacitymodel.QualityTargetMAPEDisk
		}

		report, err := capacitymodel.BuildReport(rows, capacitymodel.EstimateInput{
			Instrument:          q.Get("instrument"),
			WorkloadType:        q.Get("workload_type"),
			TargetLoad:          targetLoad,
			TargetRetentionDays: targetRetention,
			ErrorBudget:         errorBudget,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	})

	// Статические файлы: index.html, data.json и т.п.
	mux.Handle("/", http.FileServer(http.Dir(*staticDir)))

	addr := ":" + *port
	slog.Info("obs-advisor запущен", "addr", "http://localhost"+addr, "db", *dbPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

func loadRows(db *sql.DB) ([]capacitymodel.Row, error) {
	rows, err := db.Query(`
		SELECT instrument, workload_type, load_value, retention_days, duration_seconds,
		       cpu_cores, mem_peak_bytes, disk_bytes
		FROM resource_usage_info
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []capacitymodel.Row
	for rows.Next() {
		var r capacitymodel.Row
		if err := rows.Scan(
			&r.Instrument, &r.WorkloadType, &r.LoadValue, &r.RetentionDays, &r.DurationSeconds,
			&r.CPUCores, &r.MemPeakBytes, &r.DiskBytes,
		); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
