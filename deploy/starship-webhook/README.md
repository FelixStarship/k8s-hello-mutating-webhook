# cluster-agent

Command line tools cluster-agent for starship.

## TL;DR

```console
$ helm repo add --username=xxxx --password=xxxx starship-repo https://registry-chart.mypaas
$ helm install my-release starship-repo/cluster-agent
```

## Introduction

This chart bootstraps a [kubectl-tool] pod on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.12+
- Helm 2.12+ or Helm 3.0-beta3+

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ helm repo add --username=xxxx --password=xxxx helm-repo https://registry-chart.mypaas
$ helm install my-release starship-repo/cluster-agent
```

## Uninstalling the Chart

To uninstall/delete the `my-release` :

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

The following tables lists the configurable parameters of the spring-eureka-server chart and their default values per section/component:

| Parameter                   | Description                     | Default                  |
| --------------------------- | ------------------------------- | ------------------------ |
| `image.registry`            | clusterd-agent image registry    | `docker-prod-registry.cn-hangzhou.cr.aliyuncs.com` |
| `image.repository`          | clusterd-agent image name        | `cloudnative/clusterd`  |
| `image.tag`                 | clusterd-agent image tag         | `{TAG_NAME}`             |
| `arguments.hub-host`        | clusterd-agent sse svc地址 |                 |
| `arguments.hub-http`        | 回调hub地址                        | ``                    |
| `arguments.cluster`         | code值                 | ``                   |
| `namespace`                 | 应用所部署的命名空间                 | ``                   |


a YAML file that specifies the values for the parameters can be provided while installing the chart. For example.

```console
$ helm install my-release -f values.yaml helm-repo/kubectl-tool
```

\> ***\*Tip\****: You can use the default [values.yaml](values.yaml)

### 验证
```shell
> kubectl exec -it cluster-agent -n devops -- sh
>kubectl get all
NAME                        READY   STATUS    RESTARTS   AGE
pod/cluster-agent            1/1     Running   0          55m
```
### secret
kubectl get secret cloudnative-registry --output=yaml -n devops
 kubectl get secret cloudnative-registry -n devops --output="jsonpath={.data.\.dockerconfigjson}" | base64 --decode

### test
 helm install test-clusterd-agent starship-clusterd-agent  --set namespace=devops  --set arguments.hub-host=http://127.0.0.1,arguments.hub-http=http://127.0.0.1 --dry-run  -n devops