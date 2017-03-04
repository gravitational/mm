# mm

Problem - many of existing software in Kubernetes ecosystem already expose metrics in prometheus format, for example k8s apiserver or etcd.
What if you want to store metrics in InfluxDB?

Metrics middleware for Kubernetes helps with selecting services which exposes metrics by label, pull prometheus metrics from them, then convert them and push to InfluxDB.

## Requirements

* `minikube`, for installation follow [these steps](https://github.com/kubernetes/minikube#installation).
* `kubectl`, for installation follow [these steps](http://kubernetes.io/docs/getting-started-guides/kubectl/).
* `docker`, for installation follow [these steps](https://docs.docker.com/engine/installation/).
* `golang`, for installation follow [these steps](https://golang.org/doc/install).

## Quickstart

```sh
# start local k8s cluster
$ minikube start
# deploy monitoring services
$ minikube addons enable heapster
# expose InfluxDB service
$ kubectl expose service monitoring-influxdb --namespace=kube-system --type=NodePort --name influxdb
# check that you have 2 URL's
# then open first URL, connect to DB using port from second URL and then create a new database
$ minikube service influxdb --url --namespace kube-system
# deploy prometheus node exporter from https://github.com/coreos/kube-prometheus/tree/master/manifests/exporters
$ kubectl create -f kube/node-exporter-svc.yaml -f kube/node-exporter-ds.yaml
# test that it works
$ curl $(minikube ip):9100/metrics
# compile program
$ make install
# grab metrics and push them into InfluxDB
$ mm --metrics-services-label-selector=app:node-exporter --influxdb-service-namespace=kube-system --influxdb-service-name=influxdb --influxdb-database-name=<database-name>
```

For more complicated example with several metrics endpoints you may add common label to them like `metrics=true`.

## Development

Look at `Makefile` targets to know available actions. 
