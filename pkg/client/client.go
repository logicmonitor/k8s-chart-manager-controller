package client

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/constants"
	log "github.com/sirupsen/logrus"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const crdName = crv1alpha1.ChartMgrResourcePlural + "." + crv1alpha1.GroupName

// Client represents the Chart Manager client.
type Client struct {
	Clientset              *clientset.Clientset
	RESTClient             *rest.RESTClient
	APIExtensionsClientset *apiextensionsclientset.Clientset
}

// NewForConfig instantiates and returns the client and scheme.
func NewForConfig(cfg *rest.Config) (*Client, *runtime.Scheme, error) {
	s := runtime.NewScheme()
	err := crv1alpha1.AddToScheme(s)
	if err != nil {
		return nil, nil, err
	}

	client, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	config := *cfg
	config.GroupVersion = &crv1alpha1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(s)}
	restclient, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, nil, err
	}

	// Instantiate the Kubernetes API extensions client.
	apiextensionsclient, err := apiextensionsclientset.NewForConfig(&config)
	if err != nil {
		return nil, nil, err
	}

	c := &Client{
		Clientset:              client,
		RESTClient:             restclient,
		APIExtensionsClientset: apiextensionsclient,
	}

	return c, s, nil
}

// CreateCustomResourceDefinition creates the CRD for chartmgrs.
func (c *Client) CreateCustomResourceDefinition() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   crv1alpha1.GroupName,
			Version: crv1alpha1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: crv1alpha1.ChartMgrResourcePlural,
				ShortNames: []string{
					crv1alpha1.ChartMgrResourceShortNameSingular,
					crv1alpha1.ChartMgrResourceShortNamePlural,
				},
				Kind: reflect.TypeOf(crv1alpha1.ChartManager{}).Name(),
			},
			Validation: &apiextensionsv1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &apiextensionsv1beta1.JSONSchemaProps{
					Required: []string{
						"spec",
					},
					Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
						"spec": {
							Required: []string{
								"chart",
							},
							Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
								"chart": {
									Required: []string{
										"name",
									},
									Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
										"name": {
											Type:      "string",
											MinLength: func(i int64) *int64 { return &i }(1),
										},
										"repository": {
											Required: []string{
												"name",
												"url",
											},
											Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
												"name": {
													Type:      "string",
													MinLength: func(i int64) *int64 { return &i }(1),
												},
												"url": {
													Type:    "string",
													Pattern: constants.ValidateChartRepoURLPattern,
												},
											},
										},
									},
								},
								"values": {
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
													MinLength: func(i int64) *int64 { return &i }(1),
												},
												"value": {
													Type:      "string",
													MinLength: func(i int64) *int64 { return &i }(1),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// let's take a look at the json we're sending
	j, err := json.Marshal(crd)
	if err == nil {
		log.Debugf("CRD definition: %v", string(j))
	}

	log.Infof("Creating CRD %s", crdName)
	_, err = c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && strings.Contains(err.Error(), "already exists") == true {
		log.Debugf("Updating CRD %s", crdName)
		crd, err = c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		_, err = c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Update(crd)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					fmt.Printf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		log.Errorf("Error creating CRD %v", err)
		deleteErr := c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crdName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}

	log.Debugf("Created CRD")
	return crd, nil
}
