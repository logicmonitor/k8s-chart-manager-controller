package client

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	yaml "github.com/ghodss/yaml"
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

	restconfig := restConfig(cfg, s)
	restclient, err := rest.RESTClientFor(&restconfig)
	if err != nil {
		return nil, nil, err
	}

	// Instantiate the Kubernetes API extensions client.
	apiextensionsclient, err := apiextensionsclientset.NewForConfig(&restconfig)
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

func restConfig(cfg *rest.Config, s *runtime.Scheme) rest.Config {
	config := *cfg
	config.GroupVersion = &crv1alpha1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(s)}
	return config
}

// CreateCustomResourceDefinition creates the CRD for chartmgrs.
func (c *Client) CreateCustomResourceDefinition() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := c.getCRD()

	log.Infof("Creating CRD %s", crdName)
	_, err := c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return nil, err
		}
		log.Warnf("CRD %s already exists. Attempting to update.", crdName)
		return c.updateCustomResourceDefinition(crdName)
	}
	return crd, c.verify(crdName)
}

func (c *Client) verify(crdName string) error {
	err := c.waitForCRD(crdName)
	if err != nil {
		log.Errorf("Error creating CRD: %v", err)
		deleteErr := c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crdName, nil)
		if deleteErr != nil {
			return errors.NewAggregate([]error{err, deleteErr})
		}
		return err
	}
	log.Debugf("Created CRD")
	return nil
}

func (c *Client) waitForCRD(crdName string) error {
	// wait for CRD being established
	err := wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err := c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return c.checkCRDStatus(crd), err
	})
	return err
}

func (c *Client) checkCRDStatus(crd *apiextensionsv1beta1.CustomResourceDefinition) bool {
	for _, cond := range crd.Status.Conditions {
		if c.checkCondition(cond) {
			return true
		}
	}
	return false
}

func (c *Client) checkCondition(cond apiextensionsv1beta1.CustomResourceDefinitionCondition) bool {
	switch cond.Type {
	case apiextensionsv1beta1.Established:
		if cond.Status == apiextensionsv1beta1.ConditionTrue {
			return true
		}
	case apiextensionsv1beta1.NamesAccepted:
		if cond.Status == apiextensionsv1beta1.ConditionFalse {
			log.Warnf("Name conflict: %v\n", cond.Reason)
		}
	}
	return false
}

func (c *Client) getCRD() *apiextensionsv1beta1.CustomResourceDefinition {
	return &apiextensionsv1beta1.CustomResourceDefinition{
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
			Validation: constants.ChartMgrValidationRules(),
		},
	}
}

// GetCRDString returns the CRD as a YAML or JSON string
func (c *Client) GetCRDString(format string) string {
	crd := c.getCRD()

	var s []byte
	var err error

	switch format {
	case "yaml":
		s, err = yaml.Marshal(crd)
	case "json":
		s, err = json.MarshalIndent(crd, "", "  ")
	default:
		s, err = yaml.Marshal(crd)
	}
	if err != nil {
		log.Errorf("%v", err)
		return ""
	}
	return string(s)
}

func (c *Client) updateCustomResourceDefinition(crdName string) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd, err := c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	log.Debugf("Updating CRD %s", crdName)
	crd, err = c.APIExtensionsClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Update(crd)
	if err != nil {
		return nil, err
	}
	return crd, c.verify(crdName)
}
