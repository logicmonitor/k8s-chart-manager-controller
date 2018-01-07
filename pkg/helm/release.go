package lmhelm

import (
	"fmt"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/constants"
	log "github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/helm"
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

	log.Infof("Installing release %s", r.Name())
	rsp, err := r.Client.Helm.InstallReleaseFromChart(chart, r.Chartmgr.ObjectMeta.Namespace, r.installOpts()...)
	r.rls = rsp.Release
	return err
}

// Update the release
func (r *Release) Update() error {
	if r.createOnly() {
		log.Infof("CreateOnly mode. Ignoring update of release %s.", r.Name())
		return nil
	}

	chart, err := getChart(r.Chartmgr, r.Client.HelmSettings())
	if err != nil {
		return err
	}

	log.Infof("Updating release %s", r.Name())
	rsp, err := r.Client.Helm.UpdateReleaseFromChart(r.Name(), chart, r.updateOpts()...)
	r.rls = rsp.Release
	return err
}

// Delete the release
func (r *Release) Delete() error {
	if r.createOnly() {
		log.Infof("CreateOnly mode. Ignoring delete of release %s.", r.Name())
		return nil
	}

	// if the release doesn't exist, our job here is done
	if r.Name() == "" || !r.Exists() {
		log.Infof("Can't delete release %s because it doesn't exist", r.Name())
		return nil
	}

	log.Infof("Deleting release %s", r.Name())
	rsp, err := r.Client.Helm.DeleteRelease(r.Name(), r.deleteOpts()...)
	r.rls = rsp.Release
	return err
}

func (r *Release) installOpts() []helm.InstallOption {
	vals, _ := parseValues(r.Chartmgr)
	return []helm.InstallOption{
		helm.InstallReuseName(true),
		helm.InstallTimeout(r.Client.Config().ReleaseTimeoutSec),
		helm.InstallWait(true),
		helm.ReleaseName(r.Name()),
		helm.ValueOverrides(vals),
	}
}

func (r *Release) updateOpts() []helm.UpdateOption {
	vals, _ := parseValues(r.Chartmgr)
	return []helm.UpdateOption{
		helm.UpdateValueOverrides(vals),
		helm.UpgradeTimeout(r.Client.Config().ReleaseTimeoutSec),
		helm.UpgradeWait(true),
	}
}

func (r *Release) deleteOpts() []helm.DeleteOption {
	return []helm.DeleteOption{
		helm.DeletePurge(true),
		helm.DeleteTimeout(r.Client.Config().ReleaseTimeoutSec),
	}
}

func (r *Release) listOpts() []helm.ReleaseListOption {
	return []helm.ReleaseListOption{
		helm.ReleaseListFilter(r.Name()),
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
}

// StatusName returns the name of the release status
func (r *Release) Status() crv1alpha1.ChartMgrState {
	if r.rls == nil {
		return crv1alpha1.ChartMgrStateUnknown
	}
	return statusCodeToName(r.rls.Info.Status.Code)
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

func (r *Release) createOnly() bool {
	if r.Chartmgr.Spec.Options != nil && r.Chartmgr.Spec.Options.CreateOnly {
		return true
	}
	return false
}

// Deployed indicates whether or not the release is successfully deployed
func (r *Release) Deployed() bool {
	return r.rls.Info.Status.Code == rspb.Status_DEPLOYED
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
	rls, err := r.getInstalledRelease()
	if err != nil {
		log.Errorf("%v", err)
		return false
	}
	if rls == nil {
		return false
	}
	return true
}

func (r *Release) getInstalledRelease() (*rspb.Release, error) {
	// try to list the release and determine if it already exists
	log.Debugf("Attempting to locate helm release with filter %s", r.Name())
	rsp, err := r.Client.Helm.ListReleases(r.listOpts()...)
	if err != nil {
		return nil, err
	}

	if rsp.Count < 1 {
		return nil, nil
	} else if rsp.Count > 1 {
		return nil, fmt.Errorf("multiple releases found for this Chart Manager")
	}
	log.Debugf("Found helm release matching filter %s", r.Name())
	return rsp.Releases[0], nil
}
