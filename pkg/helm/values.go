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
	base := map[string]interface{}{}
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
