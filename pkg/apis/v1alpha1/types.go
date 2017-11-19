package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChartMgrState is the ChartMgr controller's state string.
type ChartMgrState string

const (
	// ChartMgrResourcePlural is the plural for the CRD.
	ChartMgrResourcePlural = "chartmanagers"
	// ChartMgrResourceShortNameSingular is the short name for the CRD.
	ChartMgrResourceShortNameSingular = "chartmgr"
	// ChartMgrResourceShortNamePlural is the short name for multiple CRDs.
	ChartMgrResourceShortNamePlural = "chartmgrs"
	// ChartMgrStateUnknown indicates that a release is in an uncertain state.
	ChartMgrStateUnknown ChartMgrState = "Unknown"
	// ChartMgrStateDeployed indicates that the release has been pushed to Kubernetes.
	ChartMgrStateDeployed ChartMgrState = "Deployed"
	// ChartMgrStateDeleted indicates that a release has been deleted from Kubermetes.
	ChartMgrStateDeleted ChartMgrState = "Deleted"
	// ChartMgrStateSuperseded indicates that this release object is outdated and a newer one exists.
	ChartMgrStateSuperseded ChartMgrState = "Superseded"
	// ChartMgrStateFailed indicates that the release was not successfully deployed.
	ChartMgrStateFailed ChartMgrState = "Failed"
	// ChartMgrStateDeleting indicates that a delete operation is underway.
	ChartMgrStateDeleting ChartMgrState = "Deleting"
	// ChartMgrStatePendingInstall indicates that an install operation is underway.
	ChartMgrStatePendingInstall ChartMgrState = "PendingInstall"
	// ChartMgrStatePendingUpgrade indicates that an upgrade operation is underway.
	ChartMgrStatePendingUpgrade ChartMgrState = "PendingUpgrade"
	// ChartMgrStatePendingRollback indicates that an rollback operation is underway.
	ChartMgrStatePendingRollback ChartMgrState = "PendingRollback"
)

// ChartManager represents the chartmgr in Kubernetes.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ChartManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ChartMgrSpec   `json:"spec,omitempty"`
	Status            ChartMgrStatus `json:"status,omitempty"`
}

// ChartMgrSpec represents the chartmgr controller's spec.
type ChartMgrSpec struct {
	Chart  *ChartMgrChart       `json:"chart,omitempty"`
	Values []*ChartMgrValuePair `json:"values,omitempty"`
}

// ChartMgrChart represents the chartmgr controller's chart definition
type ChartMgrChart struct {
	Name       string                   `json:"name,omitempty"`
	Version    string                   `json:"version,omitempty"`
	Repository *ChartMgrChartRepository `json:"repository,omitempty"`
}

// ChartMgrChartRepository represents the chartmgr controller's
// chart repository definition
type ChartMgrChartRepository struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// ChartMgrValuePair represents an chartmgr controller name/value pair
type ChartMgrValuePair struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// ChartMgrStatus is the ChartMgr controller's status.
type ChartMgrStatus struct {
	State       ChartMgrState `json:"state,omitempty"`
	ReleaseName string        `json:"release,omitempty"`
	Message     string        `json:"message,omitempty"`
}

// ChartManagerList represents a list of chartmgrs.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ChartManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ChartManager `json:"items"`
}
