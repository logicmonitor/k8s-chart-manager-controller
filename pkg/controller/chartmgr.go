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

	if !rlsExists {
		log.Infof("Release %s not found. Installing.", rlsName)
		return installRelease(chartmgr, chartmgrconfig, helmClient, rlsName, chart)
	}

	if createOnly(chartmgr) {
		log.Infof("CreateOnly mode. Ignoring update of chart %s.", rlsName)
		return getHelmRelease(helmClient, rlsName)
	}

	// if there's already a release for this chartmgr, do an upgrade.
	log.Infof("Release %s found. Updating.", rlsName)
  return updateRelease(chartmgr, chartmgrconfig, helmClient, rlsName, chart)
}

// DeleteChartMgr deletes a Chart Manager
func DeleteChartMgr(chartmgr *crv1alpha1.ChartManager, chartmgrconfig *config.Config, helmClient *helm.Client) error {

	rlsName, err := getHelmReleaseName(helmClient, string(chartmgr.ObjectMeta.UID))
	if err != nil {
		return err
	}

	if createOnly(chartmgr) {
		log.Infof("CreateOnly mode. Ignoring delete of chart %s.", rlsName)
		return nil
	}

	if rlsName != "" {
		return deleteRelease(chartmgrconfig, rlsName, helmClient)
	}
	log.Warnf("No release found for Chart Manager %s", chartmgr.ObjectMeta.UID)
	return nil
}

func removeMismatchedReleases(chartmgr *crv1alpha1.ChartManager, chartmgrconfig *config.Config, helmClient *helm.Client) error {
	// if something has previously gone wrong and there is no release associated
	// with the chartmgr, exit immediately.
	if chartmgr.Status.ReleaseName == "" {
		return nil
	}

	// check the condition wherein the calculated release name doesn't match
	// what the chartmgr thinks the name should be. this is bad.
	// we should attempt to delete the release currently associated
	// with the chartmgr.
	rlsName := getReleaseName(chartmgr)
	if releaseNamesMismatched(chartmgr, rlsName) {
		log.Warnf("Calculated release name %s does not match stored release %s", rlsName, chartmgr.Status.ReleaseName)
		n, _ := getHelmReleaseName(helmClient, chartmgr.Status.ReleaseName)
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
	if rlsName == "" {
		return false, nil
	}

	// do a lookup and see if there's already a release created for this chartmgr.
	found, err := getHelmReleaseName(helmClient, rlsName)
	if err != nil {
		return false, err
	}

	if found != "" {
		return true, nil
	}
	return false, nil
}

func createOnly(chartmgr *crv1alpha1.ChartManager) bool {
	if chartmgr.Spec.Options != nil && chartmgr.Spec.Options.CreateOnly {
		return true
	}
	return false
}

func getReleaseName(chartmgr *crv1alpha1.ChartManager) string {
	// if the release name is explicitly set, return that
	if chartmgr.Spec.Release != nil {
		log.Debugf("Release name %s specified in resource definition", chartmgr.Spec.Release.Name)
		return chartmgr.Spec.Release.Name
	}

	// releases created by the controller are formatted:
	// chartmgr-rls-[chartmgr uid]
	uid := chartmgr.ObjectMeta.UID

	rlsName := fmt.Sprintf("%s-%s", constants.ReleaseNamePrefix, uid)
	log.Debugf("Generated release name %s", rlsName)

	return rlsName
}
