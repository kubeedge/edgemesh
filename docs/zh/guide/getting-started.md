# 快速上手

## 依赖环境

[KubeEdge 所需的依赖](https://kubeedge.io/en/docs/#dependencies)

[KubeEdge v1.7+](https://github.com/kubeedge/kubeedge/releases)

::: tip
- EdgeMesh 并不依赖于 KubeEdge，它仅与标准 Kubernetes API 交互

- 鉴于边缘节点可能被割裂在不同边缘网络中的情况，我们借助于“autonomic Kube-API endpoint”功能以简化设置
:::

## Helm 安装
- **步骤1**: 修改 KubeEdge 配置

参考 [手动安装-步骤3](#step3)，修改 KubeEdge 配置。

- **步骤2**: 安装 Charts

确保你已经安装了 Helm 3

```
$ helm install edgemesh \
--set server.nodeName=<your node name> \
--set "server.advertiseAddress=<your edgemesh server adveritise address list, such as node eip>" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

server.nodeName 指定 edgemesh-server 部署的节点，server.advertiseAddress 指定 edgemesh-server 对外暴露的服务 IP 列表，多个 IP 之间使用逗号分隔，比如`{119.8.211.54,100.10.1.4}`。其中 server.advertiseAddress 是可以省略的，因为 edgemesh-server 会自动探测并配置这个列表，但不保证正确和齐全。

**例子：**

```shell
$ helm install edgemesh \
--set server.nodeName=k8s-node1 \
--set "server.advertiseAddress={119.8.211.54}" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

::: warning
请根据你的 K8s 集群设置 server.nodeName 和 server.advertiseAddress 否则 edgemesh-server 可能无法运行
:::

- **步骤3**: 检验部署结果

```shell
$ helm ls
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART           APP VERSION
edgemesh        default         1               2021-11-01 23:30:02.927346553 +0800 CST deployed        edgemesh-0.1.0  latest
```

```shell
$ kubectl get all -n kubeedge
NAME                                   READY   STATUS    RESTARTS   AGE
pod/edgemesh-agent-4rhz4               1/1     Running   0          76s
pod/edgemesh-agent-7wqgb               1/1     Running   0          76s
pod/edgemesh-agent-9c697               1/1     Running   0          76s
pod/edgemesh-server-5f6d5869ff-4568p   1/1     Running   0          5m8s

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/edgemesh-agent   3         3         3       3            3           <none>          76s

NAME                              READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/edgemesh-server   1/1     1            1           5m8s

NAME                                         DESIRED   CURRENT   READY   AGE
replicaset.apps/edgemesh-server-5f6d5869ff   1         1         1       5m8s
```

## 手动安装

- **步骤1**: 获取 EdgeMesh

```shell
$ git clone https://github.com/kubeedge/edgemesh.git
$ cd edgemesh
```

<a name="step3"></a>
- **步骤2**: 安装 CRDs

```shell
$ kubectl apply -f build/crds/istio/
customresourcedefinition.apiextensions.k8s.io/destinationrules.networking.istio.io created
customresourcedefinition.apiextensions.k8s.io/gateways.networking.istio.io created
customresourcedefinition.apiextensions.k8s.io/virtualservices.networking.istio.io created
```

- **步骤3**: 修改 KubeEdge 配置

（1）开启 Local APIServer

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
# 如果 cloudcore 没有配置为 systemd 管理，则使用如下命令重启（cloudcore 默认没有配置为 systemd 管理）
$ pkill cloudcore ; nohup /usr/local/bin/cloudcore > /var/log/kubeedge/cloudcore.log 2>&1 &

# 如果 cloudcore 配置为 systemd 管理，则使用如下命令重启
$ systemctl restart cloudcore
```

在边缘节点，打开 metaServer 模块（如果你的 KubeEdge < 1.8.0，还需关闭 edgeMesh 模块）

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

（2）配置 clusterDNS，clusterDomain

在边缘节点，配置 clusterDNS 和 clusterDomain，并重启 edgecore

```shell
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ..
  edged:
    clusterDNS: 169.254.96.16
    clusterDomain: cluster.local
..
```

```shell
$ systemctl restart edgecore
```

::: tip
clusterDNS 设置的值 '169.254.96.16' 来自于 [commonConfig](../reference/config-items.md#表1-2-commonconfig) 中 dummyDeviceIP 的默认值，如需修改请保持两者一致
:::

（3）验证

在边缘节点，测试 Local APIServer 是否开启

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

- **步骤4**: 部署 edgemesh-server

```shell
$ kubectl apply -f build/server/edgemesh/
serviceaccount/edgemesh-server created
clusterrole.rbac.authorization.k8s.io/edgemesh-server created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-server created
configmap/edgemesh-server-cfg created
deployment.apps/edgemesh-server created
```

::: warning
请根据你的 K8s 集群设置 build/server/edgemesh/04-configmap.yaml 的 advertiseAddress 和 build/server/edgemesh/05-deployment.yaml 的 nodeName，否则 edgemesh-server 可能无法运行
:::

- **步骤5**: 部署 edgemesh-agent

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/
serviceaccount/edgemesh-agent created
clusterrole.rbac.authorization.k8s.io/edgemesh-agent created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-agent created
configmap/edgemesh-agent-cfg created
daemonset.apps/edgemesh-agent created
```

- **步骤6**: 检验部署结果

```shell
$ kubectl get all -n kubeedge
NAME                                   READY   STATUS    RESTARTS   AGE
pod/edgemesh-agent-4rhz4               1/1     Running   0          76s
pod/edgemesh-agent-7wqgb               1/1     Running   0          76s
pod/edgemesh-agent-9c697               1/1     Running   0          76s
pod/edgemesh-server-5f6d5869ff-4568p   1/1     Running   0          5m8s

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/edgemesh-agent   3         3         3       3            3           <none>          76s

NAME                              READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/edgemesh-server   1/1     1            1           5m8s

NAME                                         DESIRED   CURRENT   READY   AGE
replicaset.apps/edgemesh-server-5f6d5869ff   1         1         1       5m8s
```
