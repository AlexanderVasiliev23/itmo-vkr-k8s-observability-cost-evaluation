package commands

import (
	estimate_resources "obs-bench/internal/commands/estimate-resources"
	run_experiment "obs-bench/internal/commands/run-experiment"
	experiment_usecase "obs-bench/internal/usecases/experiment"

	"github.com/spf13/cobra"
)

func NewRootCommand(experimentUsecase experiment_usecase.IExperimentUsecase) *cobra.Command {
	rootCmd := &cobra.Command{
		Short: "ObsBench — утилита для воспроизведения экспериментов по observability в Kubernetes",
		Long: `ObsBench проводит эксперименты: развёртывание системы (Prometheus, VictoriaMetrics и др.),
прогон с заданной нагрузкой, сбор замеров (RAM, CPU, диск) и вывод данных для ObsAdvisor.`,
	}
	rootCmd.AddCommand(func() *cobra.Command {
		return run_experiment.NewRunExperimentCommand(experimentUsecase, &run_experiment.Args{})
	}())
	rootCmd.AddCommand(func() *cobra.Command {
		return estimate_resources.NewEstimateResourcesCommand(&estimate_resources.Args{})
	}())

	return rootCmd
}
