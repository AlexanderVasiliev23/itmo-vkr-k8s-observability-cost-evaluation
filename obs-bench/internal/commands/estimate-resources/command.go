package estimate_resources

import (
	"encoding/json"
	"fmt"
	"obs-bench/internal/capacitymodel"
	"os"

	"github.com/spf13/cobra"
)

type Args struct {
	input               string
	instrument          string
	workloadType        string
	targetLoad          float64
	targetRetentionDays float64
	errorBudget         float64
}

func NewEstimateResourcesCommand(myArgs *Args) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate-resources",
		Short: "Оценка CPU/RAM/Disk по экспериментальным точкам",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if myArgs.input == "" || myArgs.instrument == "" || myArgs.workloadType == "" ||
				myArgs.targetLoad <= 0 || myArgs.targetRetentionDays <= 0 {
				return fmt.Errorf("required flags: --input --instrument --workload-type --target-load --target-retention-days")
			}
			if myArgs.workloadType != "metrics" && myArgs.workloadType != "logs" {
				return fmt.Errorf("--workload-type must be metrics or logs")
			}

			rows, err := capacitymodel.LoadRows(myArgs.input)
			if err != nil {
				return err
			}
			report, err := capacitymodel.BuildReport(rows, capacitymodel.EstimateInput{
				Instrument:          myArgs.instrument,
				WorkloadType:        myArgs.workloadType,
				TargetLoad:          myArgs.targetLoad,
				TargetRetentionDays: myArgs.targetRetentionDays,
				ErrorBudget:         myArgs.errorBudget,
			})
			if err != nil {
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			return enc.Encode(report)
		},
	}

	cmd.Flags().StringVar(&myArgs.input, "input", "", "Путь до data.json с measurements/model_params")
	cmd.Flags().StringVarP(&myArgs.instrument, "instrument", "i", "", "Инструмент: prometheus|victoria_metrics|loki|opensearch")
	cmd.Flags().StringVar(&myArgs.workloadType, "workload-type", "", "Тип нагрузки: metrics|logs")
	cmd.Flags().Float64Var(&myArgs.targetLoad, "target-load", 0, "Целевое load_value (например 5000)")
	cmd.Flags().Float64Var(&myArgs.targetRetentionDays, "target-retention-days", 0, "Целевой retention_days (например 3)")
	cmd.Flags().Float64Var(&myArgs.errorBudget, "error-budget", capacitymodel.QualityTargetMAPEDisk, "Диапазон ошибки (по умолчанию 0.2 = +/-20%)")

	return cmd
}
