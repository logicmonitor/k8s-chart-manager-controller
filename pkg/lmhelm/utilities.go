package lmhelm

import (
	"fmt"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
)

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
