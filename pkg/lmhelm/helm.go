package lmhelm

import (
	"fmt"

	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/config"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
)

// Client represents the LM helm client wrapper
type Client struct {
	Helm           *helm.Client
	chartmgrconfig *config.Config
	restConfig     *rest.Config
	settings       helm_env.EnvSettings
}

// Init initializes the LM helm wrapper struct
func (c *Client) Init(chartmgrconfig *config.Config, config *rest.Config) error {
	// Instantiate the Helm client
	c.chartmgrconfig = chartmgrconfig
	c.settings = c.getHelmSettings()
	c.restConfig = config

	err := c.initRepos()
	if err != nil {
		return err
	}

	c.Helm, err = c.newHelmClient()
	return err
}

// NewHeClient returns a helm client
func (c *Client) newHelmClient() (*helm.Client, error) {
	tillerHost, err := c.tillerHost()
	if err != nil {
		return nil, err
	}

	log.Infof("Using tiller host %s", tillerHost)
	heClient := helm.NewClient(helm.Host(tillerHost))
	return heClient, nil
}

func (c *Client) tillerHost() (string, error) {
	if c.settings.TillerHost != "" {
		return c.settings.TillerHost, nil
	}

	log.Debugf("Creating kubernetes client")
	client, err := kubernetes.NewForConfig(c.restConfig)
	if err != nil {
		return "", err
	}
	log.Debugf("Created kubernetes client")

	log.Debugf("Setting up port forwarding tunnel to tiller")
	tunnel, err := portforwarder.New(c.settings.TillerNamespace, client, c.restConfig)
	if err != nil {
		return "", err
	}
	log.Debugf("Set up port forwarding tunnel to tiller")

	return fmt.Sprintf("127.0.0.1:%d", tunnel.Local), nil
}

// HelmSettings returns the helm client settings
func (c *Client) HelmSettings() helm_env.EnvSettings {
	return c.settings
}

// Config returns the client application settings
func (c *Client) Config() *config.Config {
	return c.chartmgrconfig
}

func (c *Client) initRepos() error {
	err := ensureDirectories(c.settings.Home)
	if err != nil {
		return err
	}
	return ensureDefaultRepos(c.settings)
}

func (c *Client) getHelmSettings() helm_env.EnvSettings {
	var settings helm_env.EnvSettings
	settings.TillerHost = c.chartmgrconfig.TillerHost
	settings.TillerNamespace = c.chartmgrconfig.TillerNamespace
	return settings
}

func getInstalledRelease(r *Release) (*rspb.Release, error) {
	// try to list the release and determine if it already exists
	rsp, err := r.Client.Helm.ListReleases(listOpts(r)...)
	if err != nil {
		return nil, err
	}

	if rsp.Count < 1 {
		log.Debugf("Helm release %s not found", r.Name())
		return nil, nil
	} else if rsp.Count > 1 {
		return nil, fmt.Errorf("Multiple releases found for release %s", r.Name())
	}
	log.Debugf("Found helm release %s", r.Name())
	return rsp.Releases[0], nil
}

func helmInstall(r *Release, chart *chart.Chart, vals []byte) (*rspb.Release, error) {
	log.Infof("Installing release %s", r.Name())
	rsp, err := r.Client.Helm.InstallReleaseFromChart(chart, r.Chartmgr.ObjectMeta.Namespace, installOpts(r, vals)...)
	if rsp == nil || rsp.Release == nil {
		rls, _ := getInstalledRelease(r)
		if rls != nil {
			return rls, nil
		}
	} else {
		return rsp.Release, nil
	}
	return nil, err
}

func helmUpdate(r *Release, chart *chart.Chart, vals []byte) (*rspb.Release, error) {
	log.Infof("Updating release %s", r.Name())
	rsp, err := r.Client.Helm.UpdateReleaseFromChart(r.Name(), chart, updateOpts(r, vals)...)
	if rsp == nil || rsp.Release == nil {
		rls, _ := getInstalledRelease(r)
		if rls != nil {
			return rls, nil
		}
	} else {
		return rsp.Release, nil
	}
	return nil, err
}

func helmDelete(r *Release) (*rspb.Release, error) {
	log.Infof("Deleting release %s", r.Name())
	rsp, err := r.Client.Helm.DeleteRelease(r.Name(), deleteOpts(r)...)
	if rsp == nil || rsp.Release == nil {
		rls, _ := getInstalledRelease(r)
		if rls != nil {
			return rls, nil
		}
	} else {
		return rsp.Release, nil
	}
	return nil, err
}
