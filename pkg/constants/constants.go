package constants

var (
	// Version is the Chart Manager version and is set at build time.
	Version string
)

const (
	// HelmStableRepo is the name of the stable helm repo
	HelmStableRepo = "stable"
	// HelmStableRepoURL is the URL of the stable helm repo
	HelmStableRepoURL = "https://kubernetes-charts.storage.googleapis.com"
)

const (
	// ReleaseNamePrefix is the string to prepend to generated release names
	ReleaseNamePrefix = "chartmgr-rls"
)

const (
	// ChartMgrSecretName is the service account name with the proper RBAC policies to allow an chartmgr to wach the cluster.
	ChartMgrSecretName = "chartmgr"
)
