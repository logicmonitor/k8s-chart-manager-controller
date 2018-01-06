> **Note:** Chart Manager is a community driven project. LogicMonitor support will not assist in any issues related to Chart Manager.

## Chart Manager is a tool for dynamically managing Helm releases via Kubernetes custom resource objects

-  **Install, Manage, and Delete Helm releases programmatically:**
Simplify the process of installing, updating, and deleting large numbers of
Helm releases. Rather than managing all of your Helm releases out-of-band,
Chart Manager provides the ability to define and maintain all of your releases
programmatically and in-cluster.
-  **Dynamically install versioned Helm charts from public or private repositories:**
Chart Manager supports creating Helm releases from charts stored in the public
stable repository as well as custom private repositories.
**Note:** Support for repositories requiring authentication is not yet implemented
-  **Specify Helm value overrides:** In order to provide as much flexibility as
possible, Chart Manager also provides the ability to override default chart
values just as you would at the command line using '--set'.

## Chart Manager Overview
Chart Manager provides a custom controller and Kubernetes Custom Resource
Definition designed to dynamically create, manage, and delete Helm releases.
It was developed with the goal of simplifying the process required to install a
large number of applications via Helm when a new cluster is created. Chart
Manager also provides the ability to maintain a definitive list of application
deployments required for a given cluster since all releases can now be
defined and stored just like any other Kubernetes resource definition.

Chart Manager custom resource objects contain information defining a Helm
chart, Helm repository, and any optional value overrides. The Chart Manager
controller uses this information create a Helm release for the chart in the
namespace where the custom object was created. If the custom object changes,
e.g. a value override gets updated, the Chart Manager controller will attempt
to update the existing release, similar to using ```helm upgrade```. If
the custom object is deleted, the controller will delete the release.

## Chart Manager Controller Usage
```
Usage:
  k8s-chart-manager-controller [command]


Available Commands:
  crd         Dump the custom resource definition to JSON or YAML
  help        Help about any command
  manage      Start the Chart Manager controller

Flags:
      --config string   config file (default is $HOME/.k8s-chart-manager-controller.yaml)
  -h, --help            help for k8s-chart-manager-controller

Use "k8s-chart-manager-controller [command] --help" for more information about a command.
```

## Chart Manager Controller Configuration File Options
| Name              | Type   | Required | Default        | Description                                                         |
|-------------------|--------|----------|:--------------:|---------------------------------------------------------------------|
| TillerHost        | string | no       | [local tunnel] | Hostname and port of the Tiller server.                             |
| TillerNamespace   | string | no       | kube-system    | Namespace where Tiller is running.                                  |
| ReleaseTimeoutSec | int    | no       | 600            | Time in seconds to wait for a Helm release to be marked successful. |
| DebugMode         | bool   | no       | false          | Enable debug logging.                                               |

## Chart Manager Custom Object Fields
### ChartManagerSpec

| Field   | Type                     | Required | Description  |
|---------|--------------------------|----------|--------------|
| chart   | ChartManagerChart        | yes      | Helm chart configuration options. Provides information about the Helm chart to be used for creating a release. |
| release | ChartManagerRelease      | no       | Helm release configuration options. Provides information about the Helm release to be created. |
| values  | ChartManagerValue array  | no       | List of values to override in the chart. Each name/value pair is the equivalent of using the Helm CLI '--set' flag. |
| options | ChartManagerOptions      | no       | Custom object configuration options. |

### ChartManagerChart

| Field      | Type                  | Required | Description |
|------------|-----------------------|----------|-------------|
| name       | string                | yes      | Name of the chart to install. |
| version    | string                | no       | Version of the chart to install. Defaults to the latest version. |
| repository | ChartManagerChartRepo | no       | Helm chart repository configuration options. Provides the ability to install charts from a private or third-party chart repo. Defaults to stable. |

### ChartManagerRelease

| Field      | Type                  | Required | Description |
|------------|-----------------------|----------|-------------|
| name       | string                | yes      | Name of the release to create. |


### ChartManagerChartRepo
| Field     | Type   | Required | Description                       |
|-----------|--------|----------|-----------------------------------|
| name      | string | yes      | Name of the Helm chart repository |
| url       | string | yes      | URL of the Helm chart repository  |

### ChartManagerValue

| Field | Type   | Required | Description |
|-------|--------|----------|-------------|
| name  | string | yes      | Name of the value to set. Supports the same pathing and formatting options as the Helm CLI. |
| value | string | yes      | Value to assign. |

### ChartManagerOptions

| Field      | Type | Required | Description |
|------------|------|----------|-------------|
| createOnly | bool | no       | Only create the release and skip any further release management. The option is useful if you want to use Chart Manager to install a chart at cluster bootstrap but want to do ongoing management out-of-band. |

### License
[![license](https://img.shields.io/github/license/logicmonitor/k8s-argus.svg?style=flat-square)](https://github.com/logicmonitor/k8s-argus/blob/master/LICENSE)
