package lmhelm

import (
	"os"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/constants"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/utilities"
	log "github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/getter"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

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

	c := repoEntry(name, settings.Home.CacheIndex(name), url)
	r, err := createRepo(c, url, settings)
	if err != nil {
		return err
	}
	return initRepo(r, c, settings)
}

func repoEntry(name string, cache string, url string) repo.Entry {
	return repo.Entry{
		Name:  name,
		Cache: cache,
		URL:   url,
	}
}

func createRepo(c repo.Entry, url string, settings helm_env.EnvSettings) (*repo.ChartRepository, error) {
	log.Debugf("Creating chart repository from %s", url)
	return repo.NewChartRepository(&c, getter.All(settings))
}

func initRepo(r *repo.ChartRepository, c repo.Entry, settings helm_env.EnvSettings) error {
	log.Debugf("Downloading index file to %s", settings.Home.Cache())
	err := r.DownloadIndexFile("")
	if err != nil {
		return err
	}
	return initRepoFile(c, settings.Home.RepositoryFile())
}

func initRepoFile(c repo.Entry, repoFile string) error {
	// check if repo files have already been created
	_, err := os.Stat(repoFile)
	if err != nil {
		return addRepoFile(c, repoFile)
	}
	return updateRepoFile(c, repoFile)
}

func addRepoFile(c repo.Entry, repoFile string) error {
	log.Debugf("Adding repository %s", repoFile)
	f := repo.NewRepoFile()
	f.Add(&c)

	log.Debugf("Writing repository file %s", repoFile)
	return f.WriteFile(repoFile, 0644)
}

func updateRepoFile(c repo.Entry, repoFile string) error {
	log.Debugf("Loading repositories from %s", repoFile)
	f, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		return err
	}

	log.Debugf("Updating repository %s", repoFile)
	f.Update(&c)

	log.Debugf("Writing repository file %s", repoFile)
	return f.WriteFile(repoFile, 0644)
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
		err := utilities.EnsureDirectory(p)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseRepoURL(chartmgr *crv1alpha1.ChartManager) string {
	if chartmgr.Spec.Chart.Repository == nil {
		return ""
	}
	return chartmgr.Spec.Chart.Repository.URL
}

func parseRepoName(chartmgr *crv1alpha1.ChartManager) string {
	if chartmgr.Spec.Chart.Repository == nil {
		return ""
	}
	return chartmgr.Spec.Chart.Repository.Name
}
