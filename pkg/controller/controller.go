package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	chartmgrclient "github.com/logicmonitor/k8s-chart-manager-controller/pkg/client"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/config"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
)

// Controller is the Kubernetes controller object for LogicMonitor
// chartmgrs.
type Controller struct {
	*chartmgrclient.Client
	ChartMgrScheme *runtime.Scheme
	Config         *config.Config
	HelmClient     *helm.Client
	HelmSettings   helm_env.EnvSettings
}

// New instantiates and returns a Controller and an error if any.
func New(chartmgrconfig *config.Config) (*Controller, error) {
	// Instantiate the Kubernetes in cluster config.
	restconfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// Instantiate the ChartMgr client.
	client, chartmgrscheme, err := chartmgrclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	// Instantiate the Helm client.
	helmSettings := getHelmSettings(chartmgrconfig)
	err = helmInit(helmSettings)
	if err != nil {
		return nil, err
	}

	helmClient, err := newHelmClient(restconfig, helmSettings)
	if err != nil {
		return nil, err
	}

	// start a controller on instances of our custom resource
	c := &Controller{
		Client:         client,
		ChartMgrScheme: chartmgrscheme,
		Config:         chartmgrconfig,
		HelmClient:     helmClient,
		HelmSettings:   helmSettings,
	}

	return c, nil
}

// Run starts a Chart Manager resource controller.
func (c *Controller) Run(ctx context.Context) error {
	// Manage Chart Manager objects
	err := c.manage(ctx)
	if err != nil {
		return err
	}

	log.Info("Successfully started Chart Manager controller")
	<-ctx.Done()

	return ctx.Err()
}

func (c *Controller) manage(ctx context.Context) error {
	_, controller := cache.NewInformer(
		cache.NewListWatchFromClient(
			c.RESTClient,
			crv1alpha1.ChartMgrResourcePlural,
			apiv1.NamespaceAll,
			fields.Everything(),
		),
		&crv1alpha1.ChartManager{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.addFunc,
			UpdateFunc: c.updateFunc,
			DeleteFunc: c.deleteFunc,
		},
	)

	go controller.Run(ctx.Done())

	return nil
}

func (c *Controller) addFunc(obj interface{}) {
	chartmgr := obj.(*crv1alpha1.ChartManager)
	rls, err := CreateOrUpdateChartMgr(chartmgr, c.Config, c.HelmClient, c.HelmSettings)
	rlsName := ""
	if err != nil {
		log.Errorf("Failed to create Chart Manager: %v", err)
		if rls != nil {
			rlsName = rls.Name
		}
		_, updterr := c.updateChartMgrStatus(chartmgr, crv1alpha1.ChartMgrStateFailed, rlsName, err.Error())
		if updterr != nil {
			log.Warnf("Failed to update Chart Manager: %v", updterr)
			log.Errorf("Failed to create Chart Manager: %v", err)
		}
		return
	}

	status := getReleaseStatusName(rls)
	_, err = c.updateChartMgrStatus(chartmgr, status, rls.Name, string(status))
	if err != nil {
		log.Errorf("Failed to update Chart Manager status: %v", err)
		return
	}

	if err = waitForReleaseToDeploy(rls); err != nil {
		_, _ = c.updateChartMgrStatus(chartmgr, status, rls.Name, err.Error())
		log.Errorf("Failed to verify that release %v deployed: %v", rls.Name, err)
		return
	}

	log.Infof("Chart Manager %q has deployed release %q version %q",
		chartmgr.Name, chartmgr.Spec.Chart.Version, rls.Name)

	status = getReleaseStatusName(rls)
	chartmgrCopy, err := c.updateChartMgrStatus(chartmgr, status, rls.Name, string(status))
	if err != nil {
		log.Errorf("Failed to update Chart Manager status: %v", err)
		return
	}

	log.Infof("Chart Manager %q status is %q", chartmgrCopy.Name, chartmgrCopy.Status.State)
	log.Infof("Created Chart Manager: %s", chartmgrCopy.Name)
}

func (c *Controller) updateFunc(oldObj, newObj interface{}) {
	_ = oldObj.(*crv1alpha1.ChartManager)
	newChartMgr := newObj.(*crv1alpha1.ChartManager)

	_, err := CreateOrUpdateChartMgr(newChartMgr, c.Config, c.HelmClient, c.HelmSettings)
	if err != nil {
		log.Errorf("Failed to update Chart Manager: %v", err)
		return
	}

	log.Infof("Updated Chart Manager: %s", newChartMgr.Name)
}

func (c *Controller) deleteFunc(obj interface{}) {
	chartmgr := obj.(*crv1alpha1.ChartManager)

	if err := DeleteChartMgr(chartmgr, c.Config, c.HelmClient); err != nil {
		log.Errorf("Failed to delete Chart Manager: %v", err)
		return
	}

	log.Infof("Deleted Chart Manager: %s", chartmgr.Name)
}

func (c *Controller) updateChartMgrStatus(
	chartmgr *crv1alpha1.ChartManager,
	status crv1alpha1.ChartMgrState,
	rlsName string,
	message string) (*crv1alpha1.ChartManager, error) {
	chartmgrCopy := chartmgr.DeepCopy()

	log.Debugf("Updating Chart Manager status: state=%s release=%s", status, rlsName)
	chartmgrCopy.Status = crv1alpha1.ChartMgrStatus{
		State:       status,
		ReleaseName: rlsName,
		Message:     message,
	}

	err := c.RESTClient.Put().
		Name(chartmgr.ObjectMeta.Name).
		Namespace(chartmgr.ObjectMeta.Namespace).
		Resource(crv1alpha1.ChartMgrResourcePlural).
		Body(chartmgrCopy).
		Do().
		Error()

	if err != nil {
		return nil, fmt.Errorf("Failed to update status: %v", err)
	}
	return chartmgrCopy, nil
}

func getReleaseStatusName(rls *rspb.Release) crv1alpha1.ChartMgrState {
	// map the release status to our chartmgr status
	// https://github.com/kubernetes/helm/blob/8fc88ab62612f6ca81a3c1187f3a545da4ed6935/_proto/hapi/release/status.proto
	switch int32(rls.Info.Status.Code) {
	case 1:
		// Status_DEPLOYED indicates that the release has been pushed to Kubernetes.
		return crv1alpha1.ChartMgrStateDeployed
	case 2:
		// Status_DELETED indicates that a release has been deleted from Kubermetes.
		return crv1alpha1.ChartMgrStateDeleted
	case 3:
		// Status_SUPERSEDED indicates that this release object is outdated and a newer one exists.
		return crv1alpha1.ChartMgrStateSuperseded
	case 4:
		// Status_FAILED indicates that the release was not successfully deployed.
		return crv1alpha1.ChartMgrStateFailed
	case 5:
		// Status_DELETING indicates that a delete operation is underway.
		return crv1alpha1.ChartMgrStateDeleting
	case 6:
		// Status_PENDING_INSTALL indicates that an install operation is underway.
		return crv1alpha1.ChartMgrStatePendingInstall
	case 7:
		// Status_PENDING_UPGRADE indicates that an upgrade operation is underway.
		return crv1alpha1.ChartMgrStatePendingUpgrade
	case 8:
		// Status_PENDING_ROLLBACK indicates that an rollback operation is underway.
		return crv1alpha1.ChartMgrStatePendingRollback
	default:
		// Status_UNKNOWN indicates that a release is in an uncertain state.
		return crv1alpha1.ChartMgrStateUnknown
	}
}

func getReleaseStatusCode(rls *rspb.Release) rspb.Status_Code {
	return rls.Info.Status.Code
}

func releaseDeployed(rls *rspb.Release) bool {
	status := getReleaseStatusCode(rls)
	return status == rspb.Status_DEPLOYED
}

// func checkReleaseDeletedStatus(rls *rspb.Release) bool {
// 	status := getReleaseStatusCode(rls)
// 	if status == rspb.Status_DELETED {
// 		return true
// 	}
// 	return false
// }

func waitForReleaseToDeploy(rls *rspb.Release) error {
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(30 * time.Second)

	for c := ticker.C; ; <-c {
		select {
		case <-timeout:
			return errors.New("Timed out waiting for release to deploy")
		default:
			if releaseDeployed(rls) {
				return nil
			}
		}
	}
}

// func waitForReleaseToDelete(rls *rspb.Release) error {
// 	timeout := time.After(2 * time.Minute)
// 	tick := time.Tick(30 * time.Second)
//
// 	for {
// 		select {
// 		case <-timeout:
// 			return errors.New("Timed out waiting for release to delete")
// 		case <-tick:
// 			deleted := checkReleaseDeletedStatus(rls)
// 			if deleted {
// 				return nil
// 			}
// 		}
// 	}
// }
