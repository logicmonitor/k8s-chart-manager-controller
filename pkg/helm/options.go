package lmhelm

import (
	"k8s.io/helm/pkg/helm"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
)

func installOpts(r *Release, vals []byte) []helm.InstallOption {
	return []helm.InstallOption{
		helm.InstallReuseName(true),
		helm.InstallTimeout(r.Client.Config().ReleaseTimeoutSec),
		helm.InstallWait(true),
		helm.ReleaseName(r.Name()),
		helm.ValueOverrides(vals),
	}
}

func updateOpts(r *Release, vals []byte) []helm.UpdateOption {
	return []helm.UpdateOption{
		helm.UpdateValueOverrides(vals),
		helm.UpgradeTimeout(r.Client.Config().ReleaseTimeoutSec),
		helm.UpgradeWait(true),
	}
}

func deleteOpts(r *Release) []helm.DeleteOption {
	return []helm.DeleteOption{
		helm.DeletePurge(true),
		helm.DeleteTimeout(r.Client.Config().ReleaseTimeoutSec),
	}
}

func listOpts(r *Release) []helm.ReleaseListOption {
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
