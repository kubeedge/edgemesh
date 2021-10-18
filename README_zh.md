[English](./README.md) | 简体中文

# EdgeMesh

[![CI](https://github.com/kubeedge/edgemesh/actions/workflows/main.yaml/badge.svg?branch=main)](https://github.com/kubeedge/edgemesh/actions/workflows/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/edgemesh)](https://goreportcard.com/report/github.com/kubeedge/edgemesh)
[![GitHub license](https://img.shields.io/github/license/kubeedge/edgemesh)](https://github.com/kubeedge/edgemesh/blob/main/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg)](https://github.com/kubeedge/edgemesh/releases)


## 介绍

EdgeMesh 作为 KubeEdge 的一部分，为边缘场景下的服务互访提供了简单的网络方案。



#### 背景

KubeEdge基于Kubernetes构建，将云原生容器化应用程序编排能力延伸到了边缘。但是，在边缘计算场景下，网络拓扑较为复杂，不同区域中的边缘节点往往网络不互通，并且应用之间流量的互通是业务的首要需求，而EdgeMesh正是对此提供了一套解决方案。



#### 动机

EdgeMesh作为KubeEdge集群的数据面组件，为KubeEdge集群中的应用程序提供了简单的服务发现与流量代理功能，从而屏蔽了边缘场景下复杂的网络结构。



#### 优势

EdgeMesh 满足边缘场景下的新需求（如边缘资源有限，边云网络不稳定等），即实现了高可用性，高可靠性和极致轻量化：

- **高可用性**
  - 利用 KubeEdge 中的边云通道，来打通边缘节点间的网络
  - 将边缘节点间的通信分为局域网内和跨局域网
    - 局域网内的通信：直接访问
    - 跨局域网的通信：通过云端转发
- **高可靠性 （离线场景）**
  - 控制面和数据面流量都通过边云通道下发
  - EdgeMesh 内部实现轻量级的 DNS 服务器，不再访问云端 DNS
- **极致轻量化**
  - 每个节点有且仅有一个 EdgeMesh，节省边缘资源

##### 用户价值

- 对于资源受限的边缘设备，EdgeMesh 提供了一个轻量化且具有高集成度的服务发现软件
- 在现场边缘的场景下，相对于 coredns + kube-proxy + cni 这一套服务发现机制，用户只需要简单地部署一个 EdgeMesh 就能完成目标



#### 关键功能

<table align="center">
	<tr>
		<th align="center">功能</th>
		<th align="center">子功能</th>
		<th align="center">实现度</th>
	</tr >
	<tr >
		<td align="center">服务发现</td>
		<td align="center">/</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td rowspan="4" align="center">流量治理</td>
	 	<td align="center">HTTP</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">TCP</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">Websocket</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">HTTPS</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td rowspan="3" align="center">负载均衡</td>
	 	<td align="center">随机</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">轮询</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td align="center">会话保持</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td align="center">外部访问</td>
		<td align="center">/</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td align="center">多网卡监听</td>
		<td align="center">/</td>
		<td align="center">✓</td>
	</tr>
  <tr>
		<td rowspan="2" align="center">跨子网通信</td>
	 	<td align="center">跨边云通信</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">跨局域网边边通信</td>
		<td align="center">✓</td>
	</tr>
  <tr>
		<td align="center">边缘CNI</td>
	 	<td align="center">跨子网Pod通信</td>
		<td align="center">+</td>
	</tr>
</table>


**注：**

- `✓` EdgeMesh 版本所支持的功能
- `+` EdgeMesh 版本不具备的功能，但在后续版本中会支持
- `-` EdgeMesh 版本不具备的功能，或已弃用的功能



#### 未来工作
![image](./images/em-intro.png)
目前， EdgeMesh 的功能实现依赖于主机网络的连通性。未来， EdgeMesh 将会实现 CNI 插件的能力，以兼容主流 CNI 插件（例如 flannel / calico 等）的方式实现边缘节点和云上节点、跨局域网边缘节点之间的 Pod 网络连通。最终， EdgeMesh 甚至可以将部分自身组件替换成云原生组件（例如替换 [kube-proxy](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/) 实现 Cluster IP 层的能力、替换 [node local dns cache](https://kubernetes.io/docs/tasks/administer-cluster/nodelocaldns/) 实现节点级 dns 的能力、替换 [envoy](https://www.envoyproxy.io/) 实现 mesh 层的能力）。




## 架构

![image](./images/em-arch.png)
为了保证一些低版本内核、低版本 iptables 边缘设备的服务发现能力，EdgeMesh 在流量代理的实现上采用了
userspace 模式，除此之外还自带了一个轻量级的DNS解析器。 如图所示，EdgeMesh包含两个微服务： EdgeMesh-Server和Edgemesh-Agent。
EdgeMesh-Server的核心组件包括：
- **Tunnel-Server**: 基于[libp2p](https://github.com/libp2p/go-libp2p) 实现，与EdgeMesh-Agent建立连接，为EdgeMesh-Agent提供中继能力和打洞能力

EdgeMesh-Agent的核心组件包括：
- **Proxier**: 负责配置内核的iptables规则，将请求拦截到EdgeMesh进程内
- **DNS**: 内置的DNS解析器，将节点内的域名请求解析成一个服务的集群IP
- **Traffic**: 基于Go-chassis框架的流量转发模块，负责转发应用间的流量
- **Controller**: 通过KubeEdge的边缘侧list-watch能力获取Service、Endpoints、Pod等元数据
- **Tunnel-Agent**: 基于[libp2p](https://github.com/libp2p/go-libp2p) 实现，利用中继和打洞来提供跨子网通讯的能力




#### **工作原理**

- EdgeMesh-Agent 通过 KubeEdge 边缘侧 list-watch 的能力，监听Service、Endpoints等元数据的增删改，维护访问服务所需要的元数据; 同时
  配置iptables规则拦截Cluster IP网段的请求
- EdgeMesh-Agent 使用与 K8s Service 相同的 Cluster IP 和域名的方式来访问服务
- 假设我们有APP-A和APP-B两个服务，当APP-A服务基于域名访问APP-B时，域名解析请求会被本节点的EdgeMesh-Agent拦截并返回Cluster IP，这个请求会被 EdgeMesh-Agent 之前配置的
  iptables 规则重定向，转发到 EdgeMesh-Agent 进程的40001端口里（数据包从内核态->用户态）
- 请求进入 EdgeMesh-Agent 进程后，由 EdgeMesh-Agent 进程完成后端 Pod 的选择（负载均衡在这里发生），
  然后这个请求会通过tunnel模块发到APP-B所在主机的EdgeMesh-Agent上（通过中继转发或者打洞直接传输）
- App-B所在节点的EdgeMesh-Agent负责将流量转发到APP-B的服务端口上，并获取响应返回给APP-A所在节点的EdgeMesh-Agent
- APP-A所在节点的EdgeMesh-Agent负责将响应数据反馈给APP-A服务


## 入门指南
#### 预备知识
在使用EdgeMesh之前，您需要先了解以下预备知识：

- 使用 DestinationRule 时，要求 DestinationRule 的名字与相应的 Service 的名字要一致，EdgeMesh 会根据 Service 的名字来确定同命名空间下面的DestinationRule
- Service 的端口必须命名。端口名键值对必须按以下格式：name: \<protocol>[-\<suffix>]
- 使用 EdgeMesh 能力时，已经不再要求 Pod 必须开启一个 HostPort，例子可看 /examples/ 目录下面的文件

#### 获取EdgeMesh
```Shell
git clone https://github.com/kubeedge/edgemesh.git
cd edgemesh
```

#### 安装 CRDs
```shell
kubectl apply -f build/crds/istio/
```

#### 部署

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
$ pkill cloudcore
$ nohup /usr/local/bin/cloudcore > /var/log/kubeedge/cloudcore.log 2>&1 &
```

在边缘节点，查看 list-watch 是否开启

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

部署 edgemesh-server 服务

```shell
$ kubectl apply -f build/server/edgemesh/02-serviceaccount.yaml
$ kubectl apply -f build/server/edgemesh/03-clusterrole.yaml
$ kubectl apply -f build/server/edgemesh/04-clusterrolebinding.yaml
# 这里要把edgemsh-server的公网IP，也就是让边缘节点可以访问到的IP填入到05-configmap的publicIP上
$ kubectl apply -f build/server/edgemesh/05-configmap.yaml
$ kubectl apply -f build/server/edgemesh/06-deployment.yaml
```

部署 edgemesh-agent 服务

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/
namespace/kubeedge configured
serviceaccount/edgemesh-agent created
clusterrole.rbac.authorization.k8s.io/edgemesh-agent created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-agent created
configmap/edgemesh-agent-cfg created
daemonset.apps/edgemesh-agent created
```

检验部署结果

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


#### 测试样例

**HTTP协议**

在边缘节点上，部署支持 http 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/hostname.yaml
```

到边缘节点上，使用 curl 去访问相关服务，打印出容器的 hostname

```shell
$ curl hostname-lb-svc.edgemesh-test:12345
```



**TCP协议**

在边缘节点1，部署支持 tcp 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/tcp-echo-service.yaml
```

在边缘节点2，使用 telnet 去访问相关服务

```shell
$ telnet tcp-echo-service.edgemesh-test 2701
```



**Websocket协议**

在边缘节点1，部署支持 websocket 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/websocket-pod-svc.yaml
```

进入 websocket 的容器环境，并使用 client 去访问相关服务

```shell
$ docker exec -it 2a6ae1a490ae bash
$ ./client --addr ws-svc.edgemesh-test:12348
```



**负载均衡**

使用 DestinationRule 中的 loadBalancer 属性来选择不同的负载均衡模式

```shell
$ vim examples/hostname-lb-random.yaml
spec
..
  trafficPolicy:
    loadBalancer:
      simple: RANDOM
..
```



**跨边云服务发现**

处于 edgezone 的 busybox-edge 应用能够访问云上的 tcp-echo-cloud 应用，处于 cloudzone 的 busybox-cloud 能够访问边缘的 tcp-echo-edge 应用

```shell
$ kubectl apply -f examples/cloudzone.yaml
pod/tcp-echo-cloud created
service/tcp-echo-cloud-svc created
pod/busybox-sleep-cloud created

$ kubectl apply -f examples/edgezone.yaml
pod/tcp-echo-edge created
service/tcp-echo-edge-svc created
pod/busybox-sleep-edge created
```

云访问边
```shell
$ kubectl -n cloudzone exec busybox-sleep-cloud -c busybox -i -t -- sh
/ # telnet tcp-echo-edge-svc.edgezone 2701
Welcome, you are connected to node ke-edge1.
Running on Pod tcp-echo-edge.
In namespace edgezone.
With IP address 172.17.0.2.
Service default.
I'm Cloud Buxybox
I'm Cloud Buxybox
```

边访问云
```shell
$ docker exec -it 4c57a4ff8974 sh
/ # telnet tcp-echo-cloud-svc.cloudzone 2701
Welcome, you are connected to node k8s-master.
Running on Pod tcp-echo-cloud.
In namespace cloudzone.
With IP address 10.244.0.8.
Service default.
I'm Edge Busybox
I'm Edge Busybox
```



## EdgeMesh Ingress Gateway

EdgeMesh ingress gateway 提供了外部访问集群里服务的能力。

![](./images/em-ig.png)

#### HTTP网关

部署 edgemesh-gateway

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-gateway/02-configmap.yaml
configmap/edgemesh-gateway-cfg created
$ kubectl apply -f build/agent/kubernetes/edgemesh-gateway/03-deployment.yaml
deployment.apps/edgemesh-gateway created
```

创建 Gateway 资源对象和路由规则 VirtualService

```shell
$ kubectl apply -f examples/hostname-lb-random-gateway.yaml
pod/hostname-lb-edge2 created
pod/hostname-lb-edge3 created
service/hostname-lb-svc created
gateway.networking.istio.io/edgemesh-gateway configured
destinationrule.networking.istio.io/hostname-lb-edge created
virtualservice.networking.istio.io/edgemesh-gateway-svc created
```

查看 edgemesh-gateway 是否创建成功

```shell
$ kubectl get gw -n edgemesh-test
NAME               AGE
edgemesh-gateway   3m30s
```

最后，使用 IP 和 Gateway 暴露的端口来进行访问

```shell
$ curl 192.168.0.211:12345
```



#### HTTPS网关

创建测试密钥文件
```bash
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=kubeedge.io"
Generating a RSA private key
............+++++
.......................................................................................+++++
writing new private key to 'tls.key'
-----
```

根据密钥文件创建 Secret 资源对象
```bash
$ kubectl create secret tls gw-secret --key tls.key --cert tls.crt -n edgemesh-test
secret/gw-secret created
```

创建绑定了 Secret 的 Gateway 资源对象和路由规则 VirtualService
```bash
$ kubectl apply -f examples/hostname-lb-random-gateway-tls.yaml
pod/hostname-lb-edge2 created
pod/hostname-lb-edge3 created
service/hostname-lb-svc created
gateway.networking.istio.io/edgemesh-gateway configured
destinationrule.networking.istio.io/hostname-lb-edge created
virtualservice.networking.istio.io/edgemesh-gateway-svc created
```

最后，使用证书进行 HTTPS 访问
```bash
$ curl -k --cert ./tls.crt --key ./tls.key https://192.168.0.129:12345
```



## 联系方式

如果您需要支持，请从 '操作指导' 开始，然后按照我们概述的流程进行操作。

如果您有任何疑问，请通过 [KubeEdge官网](https://github.com/kubeedge/kubeedge#contact) 推荐的联系方式与我们联系

## 贡献
如果你对EdgeMesh有兴趣，希望可以为EdgeMesh做贡献，请阅读[CONTRIBUTING](./CONTRIBUTING.md)文档获取详细的贡献流程指导