# Integration with OpenShift Monitoring

https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html/monitoring/accessing-metrics#about-accessing-monitoring-web-service-apis_accessing-monitoring-apis-by-using-the-cli

```
helm install --create-namespace --namespace pvc-autoresizer pvc-autoresizer pvc-autoresizer/pvc-autoresizer --set "controller.args.prometheusURL=https://prometheus-k8s-openshift-monitoring.apps.[clustername.domain]/" --set "controller.args.bearerToken=$(oc create token cluster-monitoring-operator --duration=8760h -n openshift-monitoring)

oc get pods -n pvc-autoresizer
```

## TODO
[ ] Support proper TLS not just `InsecureSkipVerify: true`
[ ] Add support for other authentication method than bearer token
[ ] Document how to configure the `controller-pvc-autoresizer` serviceAccount for [cluster-monitoring-view](https://www.redhat.com/en/blog/custom-grafana-dashboards-red-hat-openshift-container-platform-4)
