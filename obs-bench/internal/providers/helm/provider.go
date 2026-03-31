package helm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

type IHelmProvider interface {
	Up(
		ctx context.Context,
		namespace string,
		vals map[string]interface{},
		repoURL string,
		chartName string,
		releaseName string,
	) error
	// TryUninstall ищет релиз по имени во всех namespace и удаляет его.
	// Если релиз не найден — не возвращает ошибку.
	TryUninstall(ctx context.Context, releaseName string) error
}

type provider struct{}

func NewHelmProvider() IHelmProvider {
	return &provider{}
}

func (p *provider) Up(
	ctx context.Context,
	namespace string,
	vals map[string]interface{},
	repoURL string,
	chartName string,
	releaseName string,
) error {
	settings := cli.New()
	settings.KubeConfig = os.Getenv("HOME") + "/.kube/config"
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	logger := func(format string, v ...interface{}) { slog.DebugContext(ctx, fmt.Sprintf("helm: "+format+"\n", v...)) }
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", logger); err != nil {
		return err
	}

	chartPath, err := p.pullChart(settings, repoURL, chartName)
	if err != nil {
		return err
	}
	defer os.Remove(chartPath)

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = namespace
	install.CreateNamespace = true
	install.Wait = true
	install.Timeout = 10 * time.Minute

	if _, err := install.RunWithContext(ctx, chart, vals); err != nil {
		return err
	}

	return nil
}

func (p *provider) TryUninstall(ctx context.Context, releaseName string) error {
	settings := cli.New()
	settings.KubeConfig = os.Getenv("HOME") + "/.kube/config"

	// Инициализируем без namespace чтобы листать все namespace.
	actionConfig := new(action.Configuration)
	nopLogger := func(format string, v ...interface{}) {}
	if err := actionConfig.Init(settings.RESTClientGetter(), "", "secret", nopLogger); err != nil {
		return err
	}

	list := action.NewList(actionConfig)
	list.AllNamespaces = true
	list.All = true

	releases, err := list.Run()
	if err != nil {
		return err
	}

	for _, release := range releases {
		if release.Name != releaseName {
			continue
		}

		slog.InfoContext(ctx, "uninstalling helm release", "release", releaseName, "namespace", release.Namespace)

		nsConfig := new(action.Configuration)
		if err := nsConfig.Init(settings.RESTClientGetter(), release.Namespace, "secret", nopLogger); err != nil {
			return err
		}

		uninstall := action.NewUninstall(nsConfig)
		uninstall.DisableHooks = true
		if _, err := uninstall.Run(releaseName); err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (p *provider) pullChart(settings *cli.EnvSettings, repoURL string, chartName string) (string, error) {
	// проверяем кеш
	if path, found := findChartArchive(chartName); found {
		return path, nil
	}

	pull := action.NewPullWithOpts(action.WithConfig(new(action.Configuration)))
	pull.Settings = settings
	pull.RepoURL = repoURL
	pull.DestDir = os.TempDir()
	pull.Untar = false

	if _, err := pull.Run(chartName); err != nil {
		return "", err
	}

	if path, found := findChartArchive(chartName); found {
		return path, nil
	}

	return "", fmt.Errorf("chart archive not found after pull")
}

func findChartArchive(chartName string) (string, bool) {
	dir := os.TempDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	// Только .tgz от helm pull — иначе в /tmp совпадают посторонние файлы вроде loki-helm.yaml.
	wantPrefix := chartName + "-"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".tgz") {
			continue
		}
		if strings.HasPrefix(name, wantPrefix) {
			return filepath.Join(dir, name), true
		}
	}
	return "", false
}
