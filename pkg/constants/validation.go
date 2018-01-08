package constants

import (
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/utilities"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	// ValidateChartRepoURLPattern is the regex pattern used to validate chart repository urls
	ValidateChartRepoURLPattern = "^(http:\\/\\/www\\.|https:\\/\\/www\\.|http:\\/\\/|https:\\/\\/)?[a-z0-9]+([\\-\\.]{1}[a-z0-9]+)*\\.[a-z]{2,5}(:[0-9]{1,5})?(\\/.*)?"
)

const (
	// ValidateReleaseNamePattern is the regex pattern used to validate helm release names
	ValidateReleaseNamePattern = "^[a-z0-9\\-]+?"
)

// ChartMgrValidationRules returns the CRD validation
func ChartMgrValidationRules() *apiextensionsv1beta1.CustomResourceValidation {
	return &apiextensionsv1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextensionsv1beta1.JSONSchemaProps{
			Required: []string{
				"spec",
			},
			Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
				"spec": specValidationRules(),
			},
		},
	}
}

func specValidationRules() apiextensionsv1beta1.JSONSchemaProps {
	return apiextensionsv1beta1.JSONSchemaProps{
		Required: []string{
			"chart",
		},
		Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
			"chart":   chartValidationRules(),
			"values":  valuesValidationRules(),
			"release": releaseValidationRules(),
			"options": optionsValidationRules(),
		},
	}
}

func chartValidationRules() apiextensionsv1beta1.JSONSchemaProps {
	return apiextensionsv1beta1.JSONSchemaProps{
		Required: []string{
			"name",
		},
		Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
			"name": {
				Type:      "string",
				MinLength: utilities.I64ToPI64(1),
				MaxLength: utilities.I64ToPI64(253),
			},
			"repository": repositoryValidationRules(),
		},
	}
}

func valuesValidationRules() apiextensionsv1beta1.JSONSchemaProps {
	return apiextensionsv1beta1.JSONSchemaProps{
		Type: "array",
		Items: &apiextensionsv1beta1.JSONSchemaPropsOrArray{
			Schema: &apiextensionsv1beta1.JSONSchemaProps{
				Required: []string{
					"name",
					"value",
				},
				Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
					"name": {
						Type:      "string",
						MinLength: utilities.I64ToPI64(1),
					},
					"value": {
						Type:      "string",
						MinLength: utilities.I64ToPI64(1),
					},
				},
			},
		},
	}
}

func releaseValidationRules() apiextensionsv1beta1.JSONSchemaProps {
	return apiextensionsv1beta1.JSONSchemaProps{
		Required: []string{
			"name",
		},
		Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
			"name": {
				Type:    "string",
				Pattern: ValidateReleaseNamePattern,
			},
		},
	}
}

func optionsValidationRules() apiextensionsv1beta1.JSONSchemaProps {
	return apiextensionsv1beta1.JSONSchemaProps{
		Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
			"createOnly": {
				Type: "boolean",
			},
		},
	}
}

func repositoryValidationRules() apiextensionsv1beta1.JSONSchemaProps {
	return apiextensionsv1beta1.JSONSchemaProps{
		Required: []string{
			"name",
			"url",
		},
		Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
			"name": {
				Type:      "string",
				MinLength: utilities.I64ToPI64(1),
				MaxLength: utilities.I64ToPI64(253),
			},
			"url": {
				Type:      "string",
				Pattern:   ValidateChartRepoURLPattern,
				MaxLength: utilities.I64ToPI64(2083),
			},
		},
	}
}
