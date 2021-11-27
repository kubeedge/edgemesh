# 快速上手

## 依赖环境

[KubeEdge 所需的依赖](https://kubeedge.io/en/docs/#dependencies)

[KubeEdge v1.7+](https://github.com/kubeedge/kubeedge/releases)

::: tip
EdgeMesh 依赖于 KubeEdge 的边缘 [Local APIServer](https://github.com/kubeedge/kubeedge/blob/master/CHANGELOG/CHANGELOG-1.6.md) 功能，KubeEdge v1.6+ 开始支持此功能，直到 KubeEdge v1.7+ 趋于稳定
:::

## Helm 安装

- **步骤1**: 开启 Local APIServer

参考 [手动安装-步骤3](#step3)，开启 Local APIServer。

- **步骤2**: 安装 Charts

确保你已经安装了 Helm 3

```
helm install edgemesh \
  --set server.nodeName=<your node name> \
  --set server.publicIP=<your node eip> \
  https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

server.nodeName 指定 edgemesh-server 部署的节点，server.publicIP 指定节点的公网 IP。其中 server.publicIP 是可以省略的，因为 edgemesh-server 会自动探测并配置节点的公网 IP，但不保证正确。

**例子：**

```shell
helm install edgemesh \
  --set server.nodeName=k8s-node1 \
  --set server.publicIP=119.8.211.54 \
  https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

::: warning
请根据你的 K8s 集群设置 server.nodeName 和 server.publicIP，否则 edgemesh-server 可能无法运行
:::

- **步骤3**: 检验部署结果

```shell
$ helm ls
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART           APP VERSION
edgemesh        default         1               2021-11-01 23:30:02.927346553 +0800 CST deployed        edgemesh-0.1.0  1.8.0
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
```

- **步骤3**: 开启 Local APIServer

在边缘节点，打开 metaServer 模块（如果你的 KubeEdge < 1.8.0，还需关闭 edgeMesh 模块），并重启 edgecore

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
# 如果cloudcore没有配置为systemd管理使用如下命令重启（cloudcore默认没有配置为systemd管理）
$ pkill cloudcore ; nohup /usr/local/bin/cloudcore > /var/log/kubeedge/cloudcore.log 2>&1 &

# 如果cloudcore配置为systemd管理使用如下命令重启
$ systemctl restart cloudcore
```

在边缘节点，测试 Local APIServer 是否开启

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

- **步骤4**: 部署 edgemesh-server

```shell
$ kubectl apply -f build/server/edgemesh/
namespace/kubeedge configured
serviceaccount/edgemesh-server created
clusterrole.rbac.authorization.k8s.io/edgemesh-server created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-server created
configmap/edgemesh-server-cfg created
deployment.apps/edgemesh-server created
```

::: warning
请根据你的 K8s 集群设置 05-configmap.yaml 的 publicIP 和 06-deployment.yaml 的 nodeName，否则 edgemesh-server 可能无法运行
:::

- **步骤5**: 部署 edgemesh-agent

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/
namespace/kubeedge configured
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
