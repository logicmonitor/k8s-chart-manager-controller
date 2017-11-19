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
	// HealthServerServiceName is the gRPC service name for the health checks.
	HealthServerServiceName = "grpc.health.v1.Health"
)

const (
	// ValidateChartRepoURLPattern is the regex pattern used to validate chart repository urls
	ValidateChartRepoURLPattern = "^(http:\\/\\/www\\.|https:\\/\\/www\\.|http:\\/\\/|https:\\/\\/)?[a-z0-9]+([\\-\\.]{1}[a-z0-9]+)*\\.[a-z]{2,5}(:[0-9]{1,5})?(\\/.*)?"
)
