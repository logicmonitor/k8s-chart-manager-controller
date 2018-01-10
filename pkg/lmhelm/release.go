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

func statusCodeToName(code rspb.Status_Code) crv1alpha1.ChartMgrState {
	// map the release status to our chartmgr status
	// https://github.com/kubernetes/helm/blob/8fc88ab62612f6ca81a3c1187f3a545da4ed6935/_proto/hapi/release/status.proto
	switch int32(code) {
	case 1:
		// Status_DEPLOYED indicates that the release has been pushed to Kubernetes.
		return crv1alpha1.ChartMgrStateDeployed
	case 2:
		// Status_DELETED indicates that a release has been deleted from Kubermetes.
		return crv1alpha1.ChartMgrStateDeleted
	case 3:
		// Status_SUPERSEDED indicates that this release object is outdated and a newer one exists.
		return crv1alpha1.ChartMgrStateSuperseded
	case 4:
		// Status_FAILED indicates that the release was not successfully deployed.
		return crv1alpha1.ChartMgrStateFailed
	case 5:
		// Status_DELETING indicates that a delete operation is underway.
		return crv1alpha1.ChartMgrStateDeleting
	case 6:
		// Status_PENDING_INSTALL indicates that an install operation is underway.
		return crv1alpha1.ChartMgrStatePendingInstall
	case 7:
		// Status_PENDING_UPGRADE indicates that an upgrade operation is underway.
		return crv1alpha1.ChartMgrStatePendingUpgrade
	case 8:
		// Status_PENDING_ROLLBACK indicates that an rollback operation is underway.
		return crv1alpha1.ChartMgrStatePendingRollback
	default:
		// Status_UNKNOWN indicates that a release is in an uncertain state.
		return crv1alpha1.ChartMgrStateUnknown
	}
}
