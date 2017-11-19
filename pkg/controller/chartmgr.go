package controller

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/config"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/constants"
	log "github.com/sirupsen/logrus"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/strvals"
)

// CreateOrUpdateChartMgr creates a Chart Manager
func CreateOrUpdateChartMgr(
	chartmgr *crv1alpha1.ChartManager,
	chartmgrconfig *config.Config,
	helmClient *helm.Client,
	helmSettings helm_env.EnvSettings,
	client clientset.Interface,
) (*rspb.Release, error) {

	repoURL := parseRepoURL(chartmgr)
	if repoURL != "" {
		_, err := addRepo(chartmgr.Spec.Chart.Repository.Name, repoURL, helmSettings)
		if err != nil {
			return nil, err
		}
	}

	version := parseVersion(chartmgr)
	rlsName := getReleaseName(chartmgr)
	chart, err := getChart(chartmgr.Spec.Chart.Name, version, repoURL, helmSettings)
	if err != nil {
		return nil, err
	}

	// check the condition wherein the calculated release name doesn't match
	// what the chartmgr thinks the name should be. this is bad.
	// we should attempt to delete the release currently associated
	// with the chartmgr.
	if &chartmgr.Status != nil && chartmgr.Status.ReleaseName != "" && chartmgr.Status.ReleaseName != rlsName {
		log.Warnf("Calculated release name %s does not match stored release %s", rlsName, chartmgr.Status.ReleaseName)
		n, _ := getSingleRelease(helmClient, chartmgr.Status.ReleaseName)
		if n != "" {
			err = deleteRelease(chartmgrconfig, helmClient, n)
			if err != nil {
				return nil, err
			}
		}
	}

	// do a lookup and see if there's already a release created for this chartmgr.
	foundRlsName, err := getSingleRelease(helmClient, rlsName)
	if err != nil {
		return nil, err
	}

	// if there's already a release for this chartmgr, do an upgrade.
	if foundRlsName != "" {
		log.Infof("Release %s found. Updating.", rlsName)
		rls, rlserr := updateRelease(chartmgr, chartmgrconfig, helmClient, rlsName, chart)
		if err != nil && rls != nil {
			return rls, err
		} else if err != nil {
			return nil, err
		}
		return rls, nil
	}

	log.Infof("Release %s not found. Installing.", rlsName)
	rls, err := installRelease(chartmgr, chartmgrconfig, helmClient, rlsName, chart)
	if err != nil && rls != nil {
		return rls, err
	} else if err != nil {
		return nil, err
	}
	return rls, nil
}

// DeleteChartMgr deletes aChart Manager
func DeleteChartMgr(chartmgr *crv1alpha1.ChartManager,
	chartmgrconfig *config.Config,
	helmClient *helm.Client,
	client clientset.Interface) error {

	rlsName, err := getSingleRelease(helmClient, fmt.Sprintf("%s", chartmgr.ObjectMeta.UID))
	if err != nil {
		return err
	}

	if rlsName != "" {
		delerr := deleteRelease(chartmgrconfig, helmClient, rlsName)
		if delerr != nil {
			return delerr
		}
		return nil
	}
	log.Warnf("No release found for Chart Manager %s", chartmgr.ObjectMeta.UID)
	return nil
}

func getReleaseName(chartmgr *crv1alpha1.ChartManager) string {
	// releases created by the controller are formatted:
	// chartmgr-rls-[chartmgr uid]
	uid := chartmgr.ObjectMeta.UID

	rlsName := fmt.Sprintf("%s-%s", constants.ReleaseNamePrefix, uid)
	log.Debugf("Generated release name %s", rlsName)

	return rlsName
}

func parseVersion(chartmgr *crv1alpha1.ChartManager) string {
	version := ""
	if chartmgr.Spec.Chart.Version != "" {
		version = chartmgr.Spec.Chart.Version
	}
	return version
}

func parseRepoURL(chartmgr *crv1alpha1.ChartManager) string {
	repoURL := ""
	if chartmgr.Spec.Chart.Repository != nil {
		repoURL = chartmgr.Spec.Chart.Repository.URL
	}
	return repoURL
}

func parseValues(chartmgr *crv1alpha1.ChartManager) ([]byte, error) {
	log.Debugf("Parsing values")
	base := map[string]interface{}{}
	vals := []string{}

	// iterate our name value pair and format as string
	for _, value := range chartmgr.Spec.Values {
		log.Debugf("Parsing value %s", value.Name)
		if validateValue(value) != true {
			log.Errorf("Error parsing value %v. Continuing.", value)
			continue
		}
		vals = append(vals, fmt.Sprintf("%s=%s", value.Name, value.Value))
	}

	// join k/v string and parse
	v := strings.Join(vals[:], ",")
	err := strvals.ParseInto(v, base)
	if err != nil {
		return nil, err
	}

	y, err := yaml.Marshal(base)
	if err != nil {
		return nil, err
	}

	log.Debugf("Parsed values")
	return y, nil
}

func validateValue(value *crv1alpha1.ChartMgrValuePair) bool {
	// placeholder.
	// basic type and required field validation is done at the CRD level.
	// no additional validation to be done at this time.
	return true
}
