package lmhelm

import (
	"os"
	"path/filepath"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/constants"
	log "github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
)

func getChart(chartmgr *crv1alpha1.ChartManager, settings helm_env.EnvSettings) (*chart.Chart, error) {
	err := ensureDirectories(settings.Home)
	if err != nil {
		return nil, err
	}

	url, err := getRepo(chartmgr, settings)
	if err != nil {
		return nil, err
	}

	chartFile, err := writeChart(chartmgr, url, settings)
	if err != nil {
		return nil, err
	}

	helmChart, err := loadChart(chartFile)
	if err != nil {
		return nil, err
	}
	return helmChart, nil
}

func getRepo(chartmgr *crv1alpha1.ChartManager, settings helm_env.EnvSettings) (string, error) {
	url := parseRepoURL(chartmgr)
	if url == "" {
		return constants.HelmStableRepoURL, nil
	}

	repoName := parseRepoName(chartmgr)
	err := addRepo(repoName, url, settings)
	if err != nil {
		return "", err
	}
	return url, nil
}

func writeChart(chartmgr *crv1alpha1.ChartManager, url string, settings helm_env.EnvSettings) (string, error) {
	name := chartmgr.Spec.Chart.Name
	version := parseVersion(chartmgr)

	log.Debugf("Looking for chart %s version %s in repo %s", name, version, url)

	curl, err := repo.FindChartInRepoURL(url, name, version, "", "", "", getter.All(settings))
	if err != nil {
		return "", err
	}
	log.Debugf("Chart URL found: %s", curl)

	return downloadChart(curl, version, settings)
}

func downloadChart(url string, version string, settings helm_env.EnvSettings) (string, error) {
	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Out:      os.Stdout,
		Getters:  getter.All(settings),
		Verify:   downloader.VerifyIfPossible,
	}

	log.Debugf("Downloading chart %s to %s", url, settings.Home.Archive())
	filename, _, err := dl.DownloadTo(url, version, settings.Home.Archive())
	if err != nil {
		return "", err
	}
	log.Debugf("Downloaded chart from URL %s to %s", url, filename)
	return filename, nil
}

func loadChart(filename string) (*chart.Chart, error) {
	lname, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	log.Debugf("Loading chart from %s", lname)
	chartRequested, err := chartutil.Load(lname)
	if err != nil {
		return nil, err
	}
	log.Infof("Loaded chart from %s", lname)
	return chartRequested, nil
}

func parseVersion(chartmgr *crv1alpha1.ChartManager) string {
	if chartmgr.Spec.Chart.Version == "" {
		return ""
	}
	return chartmgr.Spec.Chart.Version
}
