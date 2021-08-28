# 快速上手

## 依赖环境

[KubeEdge 所需的依赖](https://kubeedge.io/en/docs/#dependencies)

[KubeEdge v1.7+](https://github.com/kubeedge/kubeedge/releases)

::: tip
EdgeMesh 依赖于 KubeEdge 的边缘 [List-Watch](https://github.com/kubeedge/kubeedge/blob/master/CHANGELOG/CHANGELOG-1.6.md) 功能，KubeEdge v1.6+ 开始支持此功能，直到 KubeEdge v1.7+ 趋于稳定
:::

## 手动安装

- **步骤1**: 获取 EdgeMesh

```shell
$ git clone https://github.com/kubeedge/edgemesh.git
$ cd edgemesh
```

- **步骤2**: 安装 CRDs

```shell
$ kubectl apply -f build/crds/istio/
```

- **步骤3**: 开启 List-Watch

在边缘节点，关闭 edgeMesh模块，打开 metaServer 模块，并重启 edgecore

```shell
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ..
  edgeMesh:
    enable: false
  ..
  metaManager:
    metaServer:
      enable: true
..
```

```shell
$ systemctl restart edgecore
```

在云端，开启 dynamicController 模块，并重启 cloudcore

```shell
$ vim /etc/kubeedge/config/cloudcore.yaml
modules:
  ..
  dynamicController:
    enable: true
..
```

```shell
$ systemctl restart cloudcore
```

在边缘节点，测试 List-Watch 是否开启

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

- **步骤4**: 部署 edgemesh-server

```shell
$ kubectl apply -f build/server/edgemesh/02-serviceaccount.yaml
$ kubectl apply -f build/server/edgemesh/03-clusterrole.yaml
$ kubectl apply -f build/server/edgemesh/04-clusterrolebinding.yaml
# 这里要把edgemsh-server的公网IP，也就是让边缘节点可以访问到的IP填入到05-configmap.yaml的publicIP上
$ kubectl apply -f build/server/edgemesh/05-configmap.yaml
$ kubectl apply -f build/server/edgemesh/06-deployment.yaml
```

- **步骤5**: 获取集群 serviceCIDR
```shell
$ kubectl cluster-info dump | grep -m 1 service-cluster-ip-range
    "--service-cluster-ip-range=10.96.0.0/12",
```

::: tip
后续步骤需要将 serviceCIDR 填入对应的 configmap YAML。
:::

- **步骤6**: 部署 edgemesh-agent-cloud

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/03-serviceaccount.yaml
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/04-clusterrole.yaml
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/05-clusterrolebinding.yaml
# 请将06-configmap-cloud.yaml里面的subNet配置成kube-apiserver的service-cluster-ip-range的值
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/06-configmap-cloud.yaml
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/07-daemonset-cloud.yaml
```

- **步骤7**: 部署 edgemesh-agent-edge

```shell
# 请将06-configmap-edge.yaml里面的subNet配置成kube-apiserver的service-cluster-ip-range的值
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/06-configmap-edge.yaml
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/07-daemonset-edge.yaml
```

- **步骤7**: 检验部署结果

```shell
$ kubectl get all -n kubeedge
NAME                                   READY   STATUS    RESTARTS   AGE
pod/edgemesh-agent-cloud-pcphk         1/1     Running   0          19h
pod/edgemesh-agent-cloud-qkcpx         1/1     Running   0          19h
pod/edgemesh-agent-edge-b4hf7          1/1     Running   0          19h
pod/edgemesh-agent-edge-ktl6b          1/1     Running   0          19h
pod/edgemesh-server-7f97d77469-dml4j   1/1     Running   0          2d21h

NAME                                  DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/edgemesh-agent-cloud   2         2         2       2            2           <none>          19h
daemonset.apps/edgemesh-agent-edge    2         2         2       2            2           <none>          19h

NAME                              READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/edgemesh-server   1/1     1            1           2d21h

NAME                                         DESIRED   CURRENT   READY   AGE
replicaset.apps/edgemesh-server-7f97d77469   1         1         1       2d21h
```
