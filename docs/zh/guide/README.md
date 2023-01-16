# 快速上手

## 依赖环境

[KubeEdge 所需的依赖](https://kubeedge.io/en/docs/#dependencies)

[KubeEdge >= v1.7.0](https://github.com/kubeedge/kubeedge/releases)

::: tip
- EdgeMesh 并不依赖于 KubeEdge，它仅与标准 Kubernetes API 交互

- 鉴于边缘节点可能被割裂在不同边缘网络中的情况，我们借助于 [边缘 Kube-API 端点](../../zh/guide/edge-kube-api.md#快速上手) 功能以简化设置
:::

## 前置准备

- **步骤1**: 去除 K8s master 节点的污点

```shell
$ kubectl taint nodes --all node-role.kubernetes.io/master-
```
如果 K8s master 节点上没有部署需要被代理的应用，上面的步骤也可以不执行。

- **步骤2**: 给 Kubernetes API 服务添加过滤标签

```shell
$ kubectl label services kubernetes service.edgemesh.kubeedge.io/service-proxy-name=""
```

正常情况下你不会希望 EdgeMesh 去代理 Kubernetes API 服务，因此需要给它添加过滤标签，更多信息请参考 [服务过滤](../../zh/advanced/hybird-proxy.md#服务过滤)。

- **步骤3**: 启用 KubeEdge 的边缘 Kube-API 端点服务

请参考文档 [边缘 Kube-API 端点](../../zh/guide/edge-kube-api.md#快速上手) 以启用此服务。

## 安装

我们提供了两种方式安装 EdgeMesh，你可以根据自己的情况二选一去部署 EdgeMesh。

### Helm 安装

- **步骤1**: 安装 Charts

确保你已经安装了 Helm 3，然后参考：[Helm 部署 EdgeMesh 指南](https://github.com/kubeedge/edgemesh/blob/main/build/helm/edgemesh/README.md)

- **步骤2**: 检验部署结果

```shell
$ helm ls -A
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART           APP VERSION
edgemesh        kubeedge        1               2022-09-18 12:21:47.097801805 +0800 CST deployed        edgemesh-0.1.0  latest

$ kubectl get all -n kubeedge -o wide
NAME                       READY   STATUS    RESTARTS   AGE   IP              NODE         NOMINATED NODE   READINESS GATES
pod/edgemesh-agent-7gf7g   1/1     Running   0          39s   192.168.0.71    k8s-node1    <none>           <none>
pod/edgemesh-agent-fwf86   1/1     Running   0          39s   192.168.0.229   k8s-master   <none>           <none>
pod/edgemesh-agent-twm6m   1/1     Running   0          39s   192.168.5.121   ke-edge2     <none>           <none>
pod/edgemesh-agent-xwxlp   1/1     Running   0          39s   192.168.5.187   ke-edge1     <none>           <none>

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE   CONTAINERS       IMAGES                           SELECTOR
daemonset.apps/edgemesh-agent   4         4         4       4            4           <none>          39s   edgemesh-agent   kubeedge/edgemesh-agent:latest   k8s-app=kubeedge,kubeedge=edgemesh-agent
```

### 手动安装

- **步骤1**: 获取 EdgeMesh

```shell
$ git clone https://github.com/kubeedge/edgemesh.git
$ cd edgemesh
```

- **步骤2**: 安装 CRDs

```shell
$ kubectl apply -f build/crds/istio/
customresourcedefinition.apiextensions.k8s.io/destinationrules.networking.istio.io created
customresourcedefinition.apiextensions.k8s.io/gateways.networking.istio.io created
customresourcedefinition.apiextensions.k8s.io/virtualservices.networking.istio.io created
```

- **步骤3**: 部署 edgemesh-agent

```shell
$ kubectl apply -f build/agent/resources/
serviceaccount/edgemesh-agent created
clusterrole.rbac.authorization.k8s.io/edgemesh-agent created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-agent created
configmap/edgemesh-agent-cfg created
configmap/edgemesh-agent-psk created
daemonset.apps/edgemesh-agent created
```

::: tip
请根据你的 K8s 集群设置 build/agent/resources/04-configmap.yaml 的 relayNodes，并重新生成 PSK 密码。
:::

- **步骤4**: 检验部署结果

```shell
$ kubectl get all -n kubeedge -o wide
NAME                       READY   STATUS    RESTARTS   AGE   IP              NODE         NOMINATED NODE   READINESS GATES
pod/edgemesh-agent-7gf7g   1/1     Running   0          39s   192.168.0.71    k8s-node1    <none>           <none>
pod/edgemesh-agent-fwf86   1/1     Running   0          39s   192.168.0.229   k8s-master   <none>           <none>
pod/edgemesh-agent-twm6m   1/1     Running   0          39s   192.168.5.121   ke-edge2     <none>           <none>
pod/edgemesh-agent-xwxlp   1/1     Running   0          39s   192.168.5.187   ke-edge1     <none>           <none>

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE   CONTAINERS       IMAGES                           SELECTOR
daemonset.apps/edgemesh-agent   4         4         4       4            4           <none>          39s   edgemesh-agent   kubeedge/edgemesh-agent:latest   k8s-app=kubeedge,kubeedge=edgemesh-agent
```
