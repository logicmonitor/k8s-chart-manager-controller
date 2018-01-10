package lmhelm

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

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
	return r.helmInstall(chart, vals)
}

func (r *Release) helmInstall(chart *chart.Chart, vals []byte) error {
	log.Infof("Installing release %s", r.Name())
	rsp, err := r.Client.Helm.InstallReleaseFromChart(chart, r.Chartmgr.ObjectMeta.Namespace, installOpts(r, vals)...)
	if rsp == nil || rsp.Release == nil {
		rls, _ := getInstalledRelease(r)
		if rls != nil {
			r.rls = rls
		}
	} else {
		r.rls = rsp.Release
	}
	return err
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
	return r.helmUpdate(chart, vals)
}

func (r *Release) helmUpdate(chart *chart.Chart, vals []byte) error {
	log.Infof("Updating release %s", r.Name())
	rsp, err := r.Client.Helm.UpdateReleaseFromChart(r.Name(), chart, updateOpts(r, vals)...)
	if rsp == nil || rsp.Release == nil {
		rls, _ := getInstalledRelease(r)
		if rls != nil {
			r.rls = rls
		}
	} else {
		r.rls = rsp.Release
	}
	return err
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
	return r.helmDelete()
}

func (r *Release) helmDelete() error {
	log.Infof("Deleting release %s", r.Name())
	rsp, err := r.Client.Helm.DeleteRelease(r.Name(), deleteOpts(r)...)
	if rsp == nil || rsp.Release == nil {
		rls, _ := getInstalledRelease(r)
		if rls != nil {
			r.rls = rls
		}
	} else {
		r.rls = rsp.Release
	}
	return err
}
