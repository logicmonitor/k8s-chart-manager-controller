package controller

import (
	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/config"
	log "github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
)

// CreateOrUpdateChartMgr creates a Chart Manager
func CreateOrUpdateChartMgr(
	chartmgr *crv1alpha1.ChartManager,
	chartmgrconfig *config.Config,
	helmClient *helm.Client,
	helmSettings helm_env.EnvSettings,
) (*rspb.Release, error) {

	chart, err := getChart(chartmgr, helmSettings)
	if err != nil {
		return nil, err
	}

	err = removeMismatchedReleases(chartmgr, chartmgrconfig, helmClient)
	if err != nil {
		return nil, err
	}

	rlsName := getReleaseName(chartmgr)
	rlsExists, err := rlsNameExists(helmClient, rlsName)
	if err != nil {
		return nil, err
	}

	if rlsExists {
		// if there's already a release for this chartmgr, do an upgrade.
		log.Infof("Release %s found. Updating.", rlsName)
		return updateRelease(chartmgr, chartmgrconfig, helmClient, rlsName, chart)
	}

	log.Infof("Release %s not found. Installing.", rlsName)
	return installRelease(chartmgr, chartmgrconfig, helmClient, rlsName, chart)
}

// DeleteChartMgr deletes aChart Manager
func DeleteChartMgr(chartmgr *crv1alpha1.ChartManager, chartmgrconfig *config.Config, helmClient *helm.Client) error {

	rlsName, err := getSingleReleaseName(helmClient, string(chartmgr.ObjectMeta.UID))
	if err != nil {
		return err
	}

	if rlsName != "" {
		return deleteRelease(chartmgrconfig, rlsName, helmClient)
	}
	log.Warnf("No release found for Chart Manager %s", chartmgr.ObjectMeta.UID)
	return nil
}

func removeMismatchedReleases(chartmgr *crv1alpha1.ChartManager, chartmgrconfig *config.Config, helmClient *helm.Client) error {
	// check the condition wherein the calculated release name doesn't match
	// what the chartmgr thinks the name should be. this is bad.
	// we should attempt to delete the release currently associated
	// with the chartmgr.
	rlsName := getReleaseName(chartmgr)
	if releaseNamesMismatched(chartmgr, rlsName) {
		log.Warnf("Calculated release name %s does not match stored release %s", rlsName, chartmgr.Status.ReleaseName)
		n, _ := getSingleReleaseName(helmClient, chartmgr.Status.ReleaseName)
		err := deleteRelease(chartmgrconfig, n, helmClient)
		if err != nil {
			return err
		}
	}
	return nil
}

func releaseNamesMismatched(chartmgr *crv1alpha1.ChartManager, rlsName string) bool {
	if &chartmgr.Status == nil {
		return false
	}

	if chartmgr.Status.ReleaseName != rlsName {
		return true
	}
	return false
}

func rlsNameExists(helmClient *helm.Client, rlsName string) (bool, error) {
	// do a lookup and see if there's already a release created for this chartmgr.
	found, err := getSingleReleaseName(helmClient, rlsName)
	if err != nil {
		return false, err
	}

	if found != "" {
		return true, nil
	}
	return false, nil
}
