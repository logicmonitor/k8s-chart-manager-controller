package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/config"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/constants"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/utilities"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/strvals"
)

func newHelmClient(config *rest.Config, settings helm_env.EnvSettings) (*helm.Client, error) {
	if settings.TillerHost == "" {
		log.Debugf("Creating kubernetes client")
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		log.Debugf("Created kubernetes client")

		log.Debugf("Setting up port forwarding tunnel to tiller")
		tunnel, err := portforwarder.New(settings.TillerNamespace, client, config)
		if err != nil {
			return nil, err
		}
		log.Debugf("Set up port forwarding tunnel to tiller")

		settings.TillerHost = fmt.Sprintf("127.0.0.1:%d", tunnel.Local)
	}
	log.Infof("Using tiller host %s", settings.TillerHost)
	helmClient := helm.NewClient(helm.Host(settings.TillerHost))
	return helmClient, nil
}

func getHelmSettings(chartmgrconfig *config.Config) helm_env.EnvSettings {
	var settings helm_env.EnvSettings
	settings.TillerHost = chartmgrconfig.TillerHost
	settings.TillerNamespace = chartmgrconfig.TillerNamespace
	return settings
}

func helmInit(settings helm_env.EnvSettings) error {
	err := ensureDirectories(settings.Home)
	if err != nil {
		return err
	}

	return ensureDefaultRepos(settings)
}

func ensureDirectories(home helmpath.Home) error {
	configDirectories := []string{
		home.Repository(),
		home.Cache(),
		home.Plugins(),
		home.Starters(),
		home.Archive(),
	}
	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			log.Debugf("Creating directory '%s'", p)
			if err := os.MkdirAll(p, 0755); err != nil {
				return err
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s must be a directory", p)
		}
	}
	return nil
}

func ensureDefaultRepos(settings helm_env.EnvSettings) error {
	log.Debugf("Initializing stable repo %s", constants.HelmStableRepo)
	err := initStableRepo(settings)
	if err != nil {
		return err
	}
	log.Debugf("Initialized %s repo", constants.HelmStableRepo)
	return nil
}

func initStableRepo(settings helm_env.EnvSettings) error {
	return addRepo(constants.HelmStableRepo, constants.HelmStableRepoURL, settings)
}

func addRepo(name string, url string, settings helm_env.EnvSettings) error {
	if url == "" {
		return nil
	}

	repoFile := settings.Home.RepositoryFile()

	log.Debugf("Creating entry for repository %s", name)
	c := repo.Entry{
		Name:  name,
		Cache: settings.Home.CacheIndex(name),
		URL:   url,
	}
	log.Debugf("Created entry for repository %s", name)

	log.Debugf("Creating chart repository %s from %s", name, url)
	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return err
	}
	log.Debugf("Created chart repository %s from %s", name, url)

	log.Debugf("Downloading index file to %s", settings.Home.Cache())
	err = r.DownloadIndexFile("")
	if err != nil {
		return err
	}
	log.Debugf("Downloaded index file to %s", settings.Home.Cache())

	// check if repo files have already been created
	_, err = os.Stat(repoFile)
	if err == nil {
		log.Debugf("Loading repositories from %s", settings.Home.RepositoryFile())
		f, lerr := repo.LoadRepositoriesFile(settings.Home.RepositoryFile())
		if lerr != nil {
			return lerr
		}
		log.Debugf("Loaded repositories from %s", settings.Home.RepositoryFile())

		log.Debugf("Updating repository %s", settings.Home.RepositoryFile())
		f.Update(&c)
		log.Debugf("Updated repository %s", settings.Home.RepositoryFile())

		log.Debugf("Writing repository file %s", settings.Home.RepositoryFile())
		err = f.WriteFile(settings.Home.RepositoryFile(), 0644)
		if err != nil {
			return err
		}
		log.Debugf("Wrote repository file %s", settings.Home.RepositoryFile())
	} else {
		log.Debugf("Adding repository %s", settings.Home.RepositoryFile())
		f := repo.NewRepoFile()
		f.Add(&c)
		log.Debugf("Added repository %s", settings.Home.RepositoryFile())

		log.Debugf("Writing repository file %s", settings.Home.RepositoryFile())
		err = f.WriteFile(settings.Home.RepositoryFile(), 0644)
		if err != nil {
			return err
		}
		log.Debugf("Wrote repository file %s", settings.Home.RepositoryFile())
	}
	return nil
}

func getChart(chartmgr *crv1alpha1.ChartManager, settings helm_env.EnvSettings) (*chart.Chart, error) {
	name := chartmgr.Spec.Chart.Name
	repoName := parseRepoName(chartmgr)
	url := parseRepoURL(chartmgr)
	version := parseVersion(chartmgr)

	if url != "" {
		err := addRepo(repoName, url, settings)
		if err != nil {
			return nil, err
		}
	} else {
		url = constants.HelmStableRepoURL
	}

	helmchart, err := downloadChart(name, version, url, settings)
	if err != nil {
		return nil, err
	}
	return helmchart, nil
}

func downloadChart(name string, version string, url string, settings helm_env.EnvSettings) (*chart.Chart, error) {
	lver := version
	if lver == "" {
		lver = "latest"
	}
	log.Debugf("Looking for chart %s version %s in repo %s",
		name,
		lver,
		url,
	)

	url, err := repo.FindChartInRepoURL(
		url,
		name,
		version,
		"", "", "", getter.All(settings),
	)
	if err != nil {
		return nil, err
	}
	log.Debugf("Chart URL found: %s", url)
	name = url

	// TODO we should probably support TLS options in the future
	dl := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Out:      os.Stdout,
		Getters:  getter.All(settings),
		Verify:   downloader.VerifyIfPossible,
	}

	utilities.EnsureDirectory(settings.Home.Archive())
	utilities.EnsureDirectory(settings.Home.Repository())

	log.Debugf("Downloading chart %s to %s", name, settings.Home.Archive())

	filename, _, err := dl.DownloadTo(name, version, settings.Home.Archive())
	if err != nil {
		return nil, err
	}
	log.Debugf("Downloaded chart from URL %s to %s", name, filename)

	chart, err := loadChart(filename)
	if err != nil {
		return nil, err
	}
	log.Debugf("Loaded chart %s", name)
	return chart, nil
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

func installRelease(chartmgr *crv1alpha1.ChartManager, chartmgrconfig *config.Config, helmClient *helm.Client, chart *chart.Chart,
) (*rspb.Release, error) {

	vals, err := parseValues(chartmgr)
	if err != nil {
		return nil, err
	}

	rlsName := getReleaseName(chartmgr)
	ops := []helm.InstallOption{
		helm.InstallReuseName(true),
		helm.InstallTimeout(chartmgrconfig.ReleaseTimeoutSec),
		helm.InstallWait(true),
		helm.ReleaseName(rlsName),
		helm.ValueOverrides(vals),
	}

	log.Infof("Installing release %s", rlsName)
	rsp, err := helmClient.InstallReleaseFromChart(
		chart,
		chartmgr.ObjectMeta.Namespace,
		ops...,
	)
	if err != nil {
		return nil, err
	}
	log.Infof("Installed release %s", rsp.Release.Name)
	return rsp.Release, nil
}

func updateRelease(chartmgr *crv1alpha1.ChartManager, chartmgrconfig *config.Config, helmClient *helm.Client, chart *chart.Chart,
) (*rspb.Release, error) {

	rlsName := getReleaseName(chartmgr)
	rls, _ := getHelmRelease(helmClient, rlsName)

	vals, err := parseValues(chartmgr)
	if err != nil {
		return rls, err
	}

	ops := []helm.UpdateOption{
		helm.UpdateValueOverrides(vals),
		helm.UpgradeTimeout(chartmgrconfig.ReleaseTimeoutSec),
		helm.UpgradeWait(true),
	}

	log.Infof("Updating release %s", rlsName)
	rsp, err := helmClient.UpdateReleaseFromChart(rlsName, chart, ops...)
	if err != nil {
		return rls, err
	}
	log.Infof("Updated release %s", rsp.Release.Name)
	return rsp.Release, nil
}

func deleteRelease(chartmgr *crv1alpha1.ChartManager, chartmgrconfig *config.Config, helmClient *helm.Client) error {
	if rlsName == "" {
		return nil
	}

	log.Infof("Deleting release %s", rlsName)

	delOps := []helm.DeleteOption{
		helm.DeletePurge(true),
		helm.DeleteTimeout(chartmgrconfig.ReleaseTimeoutSec),
	}

	rsp, err := helmClient.DeleteRelease(rlsName, delOps...)
	if err != nil {
		return err
	}
	log.Infof("Deleted release %s", rsp.Release.Name)
	return nil
}

func getHelmReleaseName(helmClient *helm.Client, rlsFilter string) (string, error) {
	rls, err := getHelmRelease(helmClient, rlsFilter)
	if err != nil {
		log.Warnf("%s", err)
		return "", nil
	}
	if rls == nil {
		return "", nil
	}
	return rls.Name, nil
}

func getHelmRelease(helmClient *helm.Client, rlsFilter string) (*rspb.Release, error) {
	// try to list the release and determine if it already exists
	log.Debugf("Attempting to locate helm release with filter %s", rlsFilter)

	listOps := []helm.ReleaseListOption{
		helm.ReleaseListFilter(rlsFilter),
		helm.ReleaseListStatuses([]rspb.Status_Code{
			rspb.Status_DELETING,
			rspb.Status_DEPLOYED,
			rspb.Status_FAILED,
			rspb.Status_PENDING_INSTALL,
			rspb.Status_PENDING_ROLLBACK,
			rspb.Status_PENDING_UPGRADE,
			rspb.Status_UNKNOWN,
		}),
	}

	listRsp, err := helmClient.ListReleases(listOps...)
	if err != nil {
		return nil, err
	}

	if listRsp.Count < 1 {
		return nil, nil
	} else if listRsp.Count > 1 {
		return nil, fmt.Errorf("multiple releases found for this Chart Manager")
	}
	log.Debugf("Found helm release matching filter %s", rlsFilter)
	return listRsp.Releases[0], nil
}

func parseVersion(chartmgr *crv1alpha1.ChartManager) string {
	version := ""
	if chartmgr.Spec.Chart.Version != "" {
		version = chartmgr.Spec.Chart.Version
	}
	return version
}

func parseRepoURL(chartmgr *crv1alpha1.ChartManager) string {
	repoURL := ""
	if chartmgr.Spec.Chart.Repository != nil {
		repoURL = chartmgr.Spec.Chart.Repository.URL
	}
	return repoURL
}

func parseRepoName(chartmgr *crv1alpha1.ChartManager) string {
	repoName := ""
	if chartmgr.Spec.Chart.Repository != nil {
		repoName = chartmgr.Spec.Chart.Repository.Name
	}
	return repoName
}

func parseValues(chartmgr *crv1alpha1.ChartManager) ([]byte, error) {
	log.Debugf("Parsing values")
	base := map[string]interface{}{}
	vals := []string{}

	// iterate our name value pair and format as string
	for _, value := range chartmgr.Spec.Values {
		log.Debugf("Parsing value %s", value.Name)
		if !validateValue(value) {
			log.Errorf("Error parsing value %v. Continuing.", value)
			continue
		}
		vals = append(vals, fmt.Sprintf("%s=%s", value.Name, value.Value))
	}

	// join k/v string and parse
	v := strings.Join(vals[:], ",")
	err := strvals.ParseInto(v, base)
	if err != nil {
		return nil, err
	}

	y, err := yaml.Marshal(base)
	if err != nil {
		return nil, err
	}

	log.Debugf("Parsed values")
	return y, nil
}

func validateValue(value *crv1alpha1.ChartMgrValuePair) bool {
	// placeholder.
	// basic type and required field validation is done at the CRD level.
	// no additional validation to be done at this time.
	return true
}
