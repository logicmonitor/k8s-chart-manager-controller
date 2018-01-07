package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	crv1alpha1 "github.com/logicmonitor/k8s-chart-manager-controller/pkg/apis/v1alpha1"
	chartmgrclient "github.com/logicmonitor/k8s-chart-manager-controller/pkg/client"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/config"
	lmhelm "github.com/logicmonitor/k8s-chart-manager-controller/pkg/helm"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// Controller is the Kubernetes controller object for LogicMonitor
// chartmgrs.
type Controller struct {
	*chartmgrclient.Client
	ChartMgrScheme *runtime.Scheme
	Config         *config.Config
	HelmClient     *lmhelm.Client
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

	// initialize our LM helm wrapper struct
	helmClient := &lmhelm.Client{}
	err = helmClient.Init(chartmgrconfig, restconfig)
	if err != nil {
		return nil, err
	}

	// start a controller on instances of our custom resource
	c := &Controller{
		Client:         client,
		ChartMgrScheme: chartmgrscheme,
		Config:         chartmgrconfig,
		HelmClient:     helmClient,
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
	rls, err := CreateOrUpdateChartMgr(chartmgr, c.HelmClient)
	if err != nil {
		log.Errorf("Failed to create Chart Manager: %v", err)
		updterr := c.updateChartMgrStatus(chartmgr, rls, err.Error())
		if updterr != nil {
			log.Warnf("Failed to update Chart Manager: %v", updterr)
			log.Errorf("Failed to create Chart Manager: %v", err)
		}
		return
	}

	err = c.checkStatus(chartmgr, rls)
	if err != nil {
		return
	}
	log.Infof("Chart Manager %s status is %s", chartmgr.Name, chartmgr.Status.State)
	log.Infof("Created Chart Manager: %s", chartmgr.Name)
}

func (c *Controller) updateFunc(oldObj, newObj interface{}) {
	_ = oldObj.(*crv1alpha1.ChartManager)
	newChartMgr := newObj.(*crv1alpha1.ChartManager)

	rls, err := CreateOrUpdateChartMgr(newChartMgr, c.HelmClient)
	if err != nil {
		log.Errorf("Failed to update Chart Manager: %v", err)
		return
	}

	err = c.checkStatus(newChartMgr, rls)
	if err != nil {
		return
	}
	log.Infof("Updated Chart Manager: %s", newChartMgr.Name)
}

func (c *Controller) deleteFunc(obj interface{}) {
	chartmgr := obj.(*crv1alpha1.ChartManager)

	_, err := DeleteChartMgr(chartmgr, c.HelmClient)
	if err != nil {
		log.Errorf("Failed to delete Chart Manager: %v", err)
		return
	}
	log.Infof("Deleted Chart Manager: %s", chartmgr.Name)
}

func (c *Controller) checkStatus(chartmgr *crv1alpha1.ChartManager, rls *lmhelm.Release) error {
	err := c.updateChartMgrStatus(chartmgr, rls, string(rls.StatusName()))
	if err != nil {
		log.Errorf("Failed to update Chart Manager status: %v", err)
		return err
	}

	err = c.waitForReleaseToDeploy(rls)
	if err != nil {
		_ = c.updateChartMgrStatus(chartmgr, rls, err.Error())
		log.Errorf("Failed to verify that release %v deployed: %v", rls.Name, err)
		return err
	}

	log.Infof("Chart Manager %s has deployed release %s", chartmgr.Name, rls.Name())
	err = c.updateChartMgrStatus(chartmgr, rls, string(rls.StatusName()))
	if err != nil {
		log.Errorf("Failed to update Chart Manager status: %v", err)
		return err
	}
	return nil
}

func (c *Controller) updateChartMgrStatus(chartmgr *crv1alpha1.ChartManager, rls *lmhelm.Release, message string) error {

	log.Debugf("Updating Chart Manager status: state=%s release=%s", rls.StatusName(), rls.Name())
	chartmgr.Status = crv1alpha1.ChartMgrStatus{
		State:       rls.StatusName(),
		ReleaseName: rls.Name(),
		Message:     message,
	}

	err := c.RESTClient.Put().
		Name(chartmgr.ObjectMeta.Name).
		Namespace(chartmgr.ObjectMeta.Namespace).
		Resource(crv1alpha1.ChartMgrResourcePlural).
		Body(chartmgr).
		Do().
		Error()

	if err != nil {
		return fmt.Errorf("Failed to update status: %v", err)
	}
	return nil
}

func (c *Controller) waitForReleaseToDeploy(rls *lmhelm.Release) error {
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(30 * time.Second)

	for c := ticker.C; ; <-c {
		select {
		case <-timeout:
			return errors.New("Timed out waiting for release to deploy")
		default:
			if rls.Deployed() {
				return nil
			}
		}
	}
}
