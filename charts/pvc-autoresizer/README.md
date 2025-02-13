# pvc-autoresizer Helm Chart

## How to use pvc-autoresizer Helm repository

You need to add this repository to your Helm repositories:

```sh
helm repo add pvc-autoresizer https://topolvm.github.io/pvc-autoresizer
helm repo update
```

## Quick start

### Installing the Chart

To install the chart with the release name `pvc-autoresizer` using a dedicated namespace(recommended):

```sh
helm install --create-namespace --namespace pvc-autoresizer pvc-autoresizer pvc-autoresizer/pvc-autoresizer
```

Specify parameters using `--set key=value[,key=value]` argument to `helm install`.

Alternatively a YAML file that specifies the values for the parameters can be provided like this:

```sh
helm upgrade --create-namespace --namespace pvc-autoresizer -i pvc-autoresizer -f values.yaml pvc-autoresizer/pvc-autoresizer
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| cert-manager.enabled | bool | `false` | Install cert-manager together. # ref: https://cert-manager.io/docs/installation/helm/#installing-with-helm |
| controller.affinity | object | `{}` | Affinity for controller deployment. |
| controller.annotations | object | `{}` | Annotations to be added to controller deployment. |
| controller.args.additionalArgs | list | `[]` | Specify additional args. |
| controller.args.annotationPatchingEnabled | bool | `false` | Automatically patch annotations of STS provisioned PVCs to match volumeClaimTemplates. Used as "--annotation-patching-enabled" option |
| controller.args.interval | string | `"10s"` | Specify interval to monitor pvc capacity. Used as "--interval" option |
| controller.args.namespaces | list | `[]` | Specify namespaces to control the pvcs of. Empty for all namespaces. Used as "--namespaces" option |
| controller.args.prometheusURL | string | `"http://prometheus-prometheus-oper-prometheus.prometheus.svc:9090"` | Specify Prometheus URL to query volume stats. Used as "--prometheus-url" option |
| controller.args.useK8sMetricsApi | bool | `false` | Use Kubernetes metrics API instead of Prometheus. Used as "--use-k8s-metrics-api" option |
| controller.nodeSelector | object | `{}` | Map of key-value pairs for scheduling pods on specific nodes. |
| controller.podAnnotations | object | `{}` | Annotations to be added to controller pods. |
| controller.podLabels | object | `{}` | Pod labels to be added to controller pods. |
| controller.podSecurityContext | object | `{}` | Security Context to be applied to the controller pods. |
| controller.priorityClassName | string | `""` | Priority class name to be applied to the controller pods. |
| controller.replicas | int | `1` | Specify the number of replicas of the controller Pod. |
| controller.resources | object | `{"requests":{"cpu":"100m","memory":"20Mi"}}` | Specify resources. |
| controller.securityContext | object | `{}` | Security Context to be applied to the controller container within controller pods. |
| controller.terminationGracePeriodSeconds | string | `nil` | Specify terminationGracePeriodSeconds. |
| controller.tolerations | object | `{}` | Ensure pods are not scheduled on inappropriate nodes. |
| image.pullPolicy | string | `nil` | pvc-autoresizer image pullPolicy. |
| image.repository | string | `"ghcr.io/appian/pvc-autoresizer"` | pvc-autoresizer image repository to use. |
| image.tag | string | `{{ .Chart.AppVersion }}` | pvc-autoresizer image tag to use. |
| podMonitor | object | `{"additionalLabels":{},"enabled":false,"interval":"","metricRelabelings":[],"namespace":"","relabelings":[],"scheme":"http","scrapeTimeout":""}` | deploy a PodMonitor. This is not tested in CI so make sure to test it yourself. |
| podMonitor.additionalLabels | object | `{}` | Additional labels that can be used so PodMonitor will be discovered by Prometheus. |
| podMonitor.enabled | bool | `false` | If true, creates a Prometheus Operator PodMonitor. |
| podMonitor.interval | string | `""` | Interval that Prometheus scrapes metrics. |
| podMonitor.metricRelabelings | list | `[]` | MetricRelabelConfigs to apply to samples before ingestion. |
| podMonitor.namespace | string | `""` | Namespace which Prometheus is running in. |
| podMonitor.relabelings | list | `[]` | RelabelConfigs to apply to samples before scraping. |
| podMonitor.scheme | string | `"http"` | Scheme to use for scraping. |
| podMonitor.scrapeTimeout | string | `""` | The timeout after which the scrape is ended |
| serviceAccount.automountServiceAccountToken | bool | `true` | Controls the automatic mounting of ServiceAccount API credentials. |
| serviceAccount.enabled | bool | `true` | Creates a ServiceAccount for the controller deployment. |
| webhook.caBundle | string | `nil` | Specify the certificate to be used for AdmissionWebhook. |
| webhook.certificate.dnsDomain | string | `"cluster.local"` | Cluster DNS domain (required for requesting TLS certificates). |
| webhook.certificate.generate | bool | `false` | Creates a self-signed certificate for 10 years. Once the validity period has expired, simply delete the controller secret and execute helm upgrade. |
| webhook.existingCertManagerIssuer | object | `{}` | Specify the cert-manager issuer to be used for AdmissionWebhook. |
| webhook.pvcMutatingWebhook.enabled | bool | `true` | Enable PVC MutatingWebhook. |

## Generate Manifests

You can use the `helm template` command to render manifests.

```sh
helm template --namespace pvc-autoresizer pvc-autoresizer pvc-autoresizer/pvc-autoresizer
```

## Update README

The `README.md` for this chart is generated by [helm-docs](https://github.com/norwoodj/helm-docs).
To update the README, edit the `README.md.gotmpl` and generate README like below.

```console
# path to topolvm repository root
$ make setup
$ make generate-helm-docs
```

## Release Chart

pvc-autoresizer Helm Chart will be released independently.
This will prevent the pvc-autoresizer version from going up just by modifying the Helm Chart.

You must change the version of [Chart.yaml](./Chart.yaml) when making changes to the Helm Chart.
CI fails with lint error when creating a Pull Request without changing the version of [Chart.yaml](./Chart.yaml).

When you release the Helm Chart, manually run the GitHub Actions workflow for the release.

https://github.com/topolvm/pvc-autoresizer/actions/workflows/helm-release.yaml

When you run workflow, [helm/chart-releaser-action](https://github.com/helm/chart-releaser-action) will automatically create a GitHub Release.
