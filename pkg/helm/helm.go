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

	c.Helm, err = c.newHeClient()
	return err
}

// NewHeClient returns a helm client
func (c *Client) newHeClient() (*helm.Client, error) {
	if c.settings.TillerHost == "" {
		log.Debugf("Creating kubernetes client")
		client, err := kubernetes.NewForConfig(c.restConfig)
		if err != nil {
			return nil, err
		}
		log.Debugf("Created kubernetes client")

		log.Debugf("Setting up port forwarding tunnel to tiller")
		tunnel, err := portforwarder.New(c.settings.TillerNamespace, client, c.restConfig)
		if err != nil {
			return nil, err
		}
		log.Debugf("Set up port forwarding tunnel to tiller")

		c.settings.TillerHost = fmt.Sprintf("127.0.0.1:%d", tunnel.Local)
	}
	log.Infof("Using tiller host %s", c.settings.TillerHost)
	heClient := helm.NewClient(helm.Host(c.settings.TillerHost))
	return heClient, nil
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
