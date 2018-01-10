package lmhelm

import (
	"fmt"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/constants"
	log "github.com/sirupsen/logrus"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
)

// Release represents the LM helm release wrapper
type Release struct {
	Client   *Client
	Chartmgr *crv1alpha1.ChartManager
	rls      *rspb.Release
}

// Install the release
func (r *Release) Install() error {
	chart, err := getChart(r.Chartmgr, r.Client.HelmSettings())
	if err != nil {
		return err
	}

	vals, err := parseValues(r.Chartmgr)
	if err != nil {
		return err
	}
	return r.helmInstall(r, chart, vals)
}

// Update the release
func (r *Release) Update() error {
	if CreateOnly(r.Chartmgr) {
		log.Infof("CreateOnly mode. Ignoring update of release %s.", r.Name())
		return nil
	}

	log.Infof("Updating release %s", r.Name())
	chart, err := getChart(r.Chartmgr, r.Client.HelmSettings())
	if err != nil {
		return err
	}

	vals, err := parseValues(r.Chartmgr)
	if err != nil {
		return err
	}
	return r.helmUpdate(r, chart, vals)
}

// Delete the release
func (r *Release) Delete() error {
	if CreateOnly(r.Chartmgr) {
		log.Infof("CreateOnly mode. Ignoring delete of release %s.", r.Name())
		return nil
	}

	// if the release doesn't exist, our job here is done
	if r.Name() == "" || !r.Exists() {
		log.Infof("Can't delete release %s because it doesn't exist", r.Name())
		return nil
	}
	return r.helmDelete(r)
}

// Status returns the name of the release status
func (r *Release) Status() crv1alpha1.ChartMgrState {
	if r.rls == nil || r.rls.Info == nil || r.rls.Info.Status == nil {
		return crv1alpha1.ChartMgrStateUnknown
	}
	return statusCodeToName(r.rls.Info.Status.Code)
}

// CreateOnly returns true of the chart manager CreateOnly option is set
func CreateOnly(chartmgr *crv1alpha1.ChartManager) bool {
	if chartmgr.Spec.Options != nil && chartmgr.Spec.Options.CreateOnly {
		return true
	}
	return false
}

// Deployed indicates whether or not the release is successfully deployed
func (r *Release) Deployed() bool {
	rls, err := getInstalledRelease(r)
	if err != nil {
		log.Errorf("%v", err)
		return false
	}
	if rls == nil || rls.Info == nil || rls.Info.Status == nil {
		return false
	}
	r.rls = rls
	return rls.Info.Status.Code == rspb.Status_DEPLOYED
}

// Name returns the name of this release
func (r *Release) Name() string {
	// if the release name is explicitly set, return that
	if r.Chartmgr.Spec.Release != nil {
		// log.Debugf("Release name %s specified in resource definition", r.Chartmgr.Spec.Release.Name)
		return r.Chartmgr.Spec.Release.Name
	}

	// releases created by the controller are formatted:
	// chartmgr-rls-[chartmgr uid]
	uid := r.Chartmgr.ObjectMeta.UID

	return fmt.Sprintf("%s-%s", constants.ReleaseNamePrefix, uid)
}

// Exists indicates whether or not the release exists in-cluster
func (r *Release) Exists() bool {
	rls, err := getInstalledRelease(r)
	if err != nil {
		log.Errorf("%v", err)
		return false
	}
	if rls == nil {
		return false
	}
	r.rls = rls
	return true
}
