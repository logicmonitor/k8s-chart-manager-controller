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
	rspb "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
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

func parseReleaseFromResponse(r *Release, v interface{}) *rspb.Release {
	rls := getReleaseFromMessage(v)
	if v == nil || rls == nil {
		rls, _ := r.getInstalledRelease()
		if rls != nil {
			return rls
		}
	} else {
		return rls
	}
	return nil
}

func getReleaseFromMessage(v interface{}) *rspb.Release {
	if v == nil {
		return nil
	}

	switch v.(type) {
	case *services.InstallReleaseResponse:
		m := v.(*services.InstallReleaseResponse)
		return m.Release
	case *services.UpdateReleaseResponse:
		m := v.(*services.UpdateReleaseResponse)
		return m.Release
	case *services.UninstallReleaseResponse:
		m := v.(*services.UninstallReleaseResponse)
		return m.Release
	default:
		return nil
	}
}
