package lmhelm

import (
	"fmt"
	"strings"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/strvals"
)

func parseValues(chartmgr *crv1alpha1.ChartManager) ([]byte, error) {
	log.Debugf("Parsing values")

	s := valuesToString(chartmgr)
	y, err := stringValuesToYaml(s)
	if err != nil {
		return nil, err
	}

	log.Debugf("Parsed values")
	return y, nil
}

func valuesToString(chartmgr *crv1alpha1.ChartManager) string {
	vals := []string{}

	// iterate our name value pair and format as string
	for _, value := range chartmgr.Spec.Values {
		log.Debugf("Parsing value %s", value.Name)
		if !validateValue(value) {
			log.Errorf("Error parsing value %v. Continuing.", value)
			continue
		}
		vals = append(vals, fmt.Sprintf("%s=%s", value.Name, value.Value))
	}
	return strings.Join(vals[:], ",")
}

func stringValuesToYaml(s string) ([]byte, error) {
	base := map[string]interface{}{}
	// join k/v string and parse
	err := strvals.ParseInto(s, base)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(base)
}

func validateValue(value *crv1alpha1.ChartMgrValuePair) bool {
	// placeholder.
	// basic type and required field validation is done at the CRD level.
	// no additional validation to be done at this time.
	return true
}
