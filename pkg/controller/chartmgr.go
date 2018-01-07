package controller

import (
	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	lmhelm "github.com/logicmonitor/k8s-chart-manager-controller/pkg/helm"
	log "github.com/sirupsen/logrus"
)

// CreateOrUpdateChartMgr creates a Chart Manager
func CreateOrUpdateChartMgr(chartmgr *crv1alpha1.ChartManager, client *lmhelm.Client) (*lmhelm.Release, error) {
	rls := &lmhelm.Release{
		Client:   client,
		Chartmgr: chartmgr,
	}

	err := removeMismatchedReleases(chartmgr, rls)
	if err != nil {
		return nil, err
	}

	if rls.Exists() {
		log.Infof("Release %s found. Updating.", rls.Name())
		err = rls.Update()
		if err != nil {
			return nil, err
		}
		return rls, err
	}
	log.Infof("Release %s not found. Installing.", rls.Name())
	err = rls.Install()
	if err != nil {
		return nil, err
	}
	return rls, err
}

// DeleteChartMgr deletes a Chart Manager
func DeleteChartMgr(chartmgr *crv1alpha1.ChartManager, client *lmhelm.Client) (*lmhelm.Release, error) {
	rls := &lmhelm.Release{
		Client:   client,
		Chartmgr: chartmgr,
	}
	err := rls.Delete()
	if err != nil {
		return nil, err
	}
	return rls, err
}

func removeMismatchedReleases(chartmgr *crv1alpha1.ChartManager, rls *lmhelm.Release) error {
	// check the condition wherein the calculated release name doesn't match
	// what the chartmgr thinks the name should be. this is bad.
	// we should attempt to delete the release currently associated
	// with the chartmgr.
	if resourceReleaseName(chartmgr) == rls.Name() {
		log.Warnf("Calculated release name %q does not match stored release %q", rls.Name(), resourceReleaseName(chartmgr))
		return rls.Delete()
	}
	return nil
}

func resourceReleaseName(chartmgr *crv1alpha1.ChartManager) string {
	if &chartmgr.Status == nil {
		return ""
	}
	return chartmgr.Status.ReleaseName
}
