# 快速上手

## 依赖环境

[KubeEdge 所需的依赖](https://kubeedge.io/en/docs/#dependencies)

[KubeEdge >= v1.7.0](https://github.com/kubeedge/kubeedge/releases)

::: tip
- EdgeMesh 并不依赖于 KubeEdge，它仅与标准 Kubernetes API 交互

- 鉴于边缘节点可能被割裂在不同边缘网络中的情况，我们借助于“autonomic Kube-API endpoint”功能以简化设置
:::

## 前置准备
- **步骤1**: 去除 K8s master 节点的污点

```
kubectl taint nodes --all node-role.kubernetes.io/master-
```
如果 K8s master 节点上没有部署需要被代理的应用，上面的步骤也可以不执行。

- **步骤2**: 修改 KubeEdge 配置

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
# 如果 cloudcore 没有配置为 systemd 管理，则使用如下命令重启（cloudcore < 1.10 默认没有配置为 systemd 管理）
$ pkill cloudcore ; nohup /usr/local/bin/cloudcore > /var/log/kubeedge/cloudcore.log 2>&1 &

# 如果 cloudcore 配置为 systemd 管理，则使用如下命令重启
$ systemctl restart cloudcore

# 如果 cloudcore 使用容器化部署，则用 kubectl 删除 cloudcore 的 pod
$ kubectl -n kubeedge delete pod <your cloudcore pod name>
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
clusterDNS 设置的值 '169.254.96.16' 来自于 [commonConfig](https://edgemesh.netlify.app/zh/reference/config-items.html#edgemesh-agent-cfg) 中 bridgeDeviceIP 的默认值，如需修改请保持两者一致
:::

（3）验证

在边缘节点，测试 Local APIServer 是否正常开启

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

::: warning
如果返回值是空列表，或者响应时长很久（接近 10s）才拿到返回值，说明你的配置可能有误，请再检查一次。
:::

## Helm 安装

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

## 手动安装

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
