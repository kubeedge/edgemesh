# EdgeMesh 高可用特性


​EdgeMesh高可用特性主要针对分布式动态中继连接和私有局域网的网络自治等边缘场景，为节点提供中继流量以及打洞连接服务，保障集群连接能在边缘场景当中始终通畅。



## 高可用特性使用指南

### 基本原理介绍

高可用特性将 edgemesh-server 的能力合并到了 edgemesh-agent 的 EdgeTunnel 模块之中，使得具备中继能力的 edgemesh-agent 能够自动成为中继服务器，为其他节点提供内网穿透和中继转发的功能，新老系统架构对比如下：


![img](/images/arch.png)


这一项特性的主要实现原理是：当集群内有节点具备中继能力时，其上的 edgemesh-agent 会承担起中继节点的角色，来为其他节点提供内网穿透和流量中继转发的服务。在集群初始化或者是有节点新加入集群时，EdgeMesh 会基于 mDNS 机制发现局域网内的节点并作记录，同时 DHT 机制会响应跨局域网其他节点发来的连接请求并作记录。这样当集群内跨局域网的两节点需要连接的时候，中继节点就可以为它们提供流量中继和协助内网穿透的服务。



![img](/images/linkBreak.png)

EdgeMesh 高可用特性的核心功能如上图所示，集群中 A 节点与 B 节点通过 R1 中继节点连接来提供服务。当 R1 节点无法提供中继服务的时候，A、B 节点可以通过高可用特性自动切换到中继节点 R2 并重新建立连接。在这个过程当中用户几乎感受不到网络连接的变化。

以下我将简单介绍不同情况下使用 EdgeMesh 高可用特性的方式。

### 部署时启用高可用特性

您可以通过以下配置方法，在**安装EdgeMesh**时启用高可用特性，配置过程当中您可以依据集群连接的需求配置中继节点的地址：

```
# 启用高可用特性
helm install edgemesh --namespace kubeedge \
--set agent.relayNodes[0].nodeName=k8s-master,agent.relayNodes[0].advertiseAddress="{1.1.1.1}" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

+ `relayNodes` 参数是中继节点表，类型为 `[]relayNode`，您可以通过配置它来指定集群中应该承担中继节点角色的 edgemesh-agent。

+ `relayNode.nodeName` 参数使用节点名的方式来指定 relay 节点，这必须与K8s的节点名相同，您可以通过 `kubectl get nodes` 查看您的k8s节点名。

+ `relayNode.advertiseAddress` 参数用于指定 relay 节点的地址，其应当与节点在K8s集群当中的节点地址一致, 如果您购买了公有云的公网IP并挂载到此 relay 节点上，则 `relayNode.advertiseAddress` 参数最好应该填写该公网IP地址。

| **名称**                      | **参数类型** | **使用示例**                                           | **功能描述**                                                 |
| ----------------------------- | ------------ | ------------------------------------------------------ | ------------------------------------------------------------ |
| relayNodes[].nodeName         | string       | --set agent.relayNodes[0].nodeName=k8s-master          | 设置中继节点的节点名称，应当与其在K8S集群当中设置的nodeName一致 |
| relayNodes[].advertiseAddress | []string     | --set agent.relayNodes[0].advertiseAddress="{1.1.1.1}" | 设置中继节点的节点地址，应当与其在K8S集群当中的节点地址一致  |

需要注意的是：设置中继节点的数量由 `relayNodes[num]` 中索引值 num 来规定，num 取值从 0 开始，`relayNodes[0]` 表示中继节点1。

更多的安装配置信息请详见：

helm安装 :  [Helm 安装 | EdgeMesh](https://edgemesh.netlify.app/zh/guide/#helm-安装)

手动安装  :  [快速上手 | EdgeMesh](https://edgemesh.netlify.app/zh/guide/)



### 运行时添加新中继节点

如果您在**使用EdgeMesh**高可用特性时，想要在集群当中添加新的中继节点，可以通过修改 `edgemesh-agent-cfg` 当中的 `relayNodes `参数来达到目的，以下为具体修改配置的方式：



```
kubectl -n kubeedge edit configmap edgemesh-agent-cfg

# 进入config 文件当中进行编辑
apiVersion: v1
data:
 edgemesh-agent.yaml: |-
   modules:
     edgeProxy:
       enable: true
     edgeTunnel:
       enable: true
       # 设置添加或者是修改为新的中继节点
       relayNodes:
       - nodeName: R1
         advertiseAddress:
         - 1.1.1.1
       - nodeName: R2   <------  在此配置新增节点
         advertiseAddress:
         - 192.168.5.103
```





之后您可以使用` kubeadm join `或者` keadm join `添加新的中继节点 R2，接着通过以下操作查看添加的中继节点是否正常运行：



```
# 查看节点是否正常添加
kubectl get nodes
NAME         STATUS   ROLES                  AGE    VERSION
k8s-master   Ready    control-plane,master   249d   v1.21.8
k8s-node1    Ready    <none>                 249d   v1.21.8
ke-edge1     Ready    agent,edge             234d   v1.19.3-kubeedge-v1.8.2
ke-edge2     Ready    agent,edge             5d     v1.19.3-kubeedge-v1.8.2
R2           Ready    agent,edge             1d     v1.19.3-kubeedge-v1.8.2 <------ 新节点


# 查看中继节点的 edgemesh-agent 是否正常运行
kubectl get all -n kubeedge -o wide
NAME                       READY   STATUS    RESTARTS   AGE   IP              NODE         NOMINATED NODE   READINESS GATES
pod/edgemesh-agent-59fzk   1/1     Running   0          10h   192.168.5.187   ke-edge1     <none>           <none>
pod/edgemesh-agent-hfsmz   1/1     Running   1          10h   192.168.0.229   k8s-master   <none>           <none>
pod/edgemesh-agent-tvhks   1/1     Running   0          10h   192.168.0.71    k8s-node1    <none>           <none>
pod/edgemesh-agent-tzntc   1/1     Running   0          10h   192.168.5.121   ke-edge2     <none>           <none>
pod/edgemesh-agent-kasju   1/1     Running   0          10h   192.168.5.103   R2           <none>           <none> <------ new edgemesh-agent running on R2
```



### 运行时转化节点成中继节点

如果您**在集群运行过程**当中，想要将一些已有节点转化为中继节点，只需要修改 `edgemesh-agent-cfg` 当中的 `relayNodes `参数即可, 以下为具体修改配置的方式：



```
kubectl -n kubeedge edit configmap edgemesh-agent-cfg

# 进入config 文件当中进行编辑
apiVersion: v1
data:
 edgemesh-agent.yaml: |-
   modules:
     edgeProxy:
       enable: true
     edgeTunnel:
       enable: true
       # 设置新的中继节点
       relayNodes:
       - nodeName: R1
         advertiseAddress:
         - 1.1.1.1
       - nodeName: R2   <------  在此配置新节点信息
         advertiseAddress:
         - 192.168.5.103
```



修改完此配置后，**需要重启R2节点（转化节点）上的edgemesh-agent**其余节点能自动发现此新的中继节点。

+ edgemesh-agent 会读取 configmap 里的中继节点表 relayNodes，检查自己是否被用户设置为中继节点。如果在relayNodes中读取到 R2 存在，则表明 R2 被设置为默认初始的中继节点。

+ R2节点上的 edgemesh-agent 会尝试成为relay ，同时启动对应的中继功能。

+ 如果发现该节点没有中继能力（一般挂载了公网IP的节点会具备中继能力），那么该节点还是不能承担起中继节点的角色（它还是会以普通节点的模式工作）。



## 高可用特性应用场景

​高可用架构的主要目的是为了保障系统的稳定性以及提升系统的整体性能，此次 EdgeMesh 高可用特性在原有功能的基础上还覆盖了多种边缘网络的痛点场景。以下为 EdgeMesh 高可用特性在边缘计算场景下的具体应用场景，用户可以依据这些用例来理解本特性能提供的服务。

### 单点故障以及高负载场景

​如图所示，当单个节点承担中继功能时，所有其他的节点都需要连接该节点才能够获取网络连接的服务。在这样的场景当中，单个节点的负载就会相应地增加，过高的通信负载或者是密集的连接数量，在诸多情况下成为限制服务性能的主要原因，同时如果该节点出现故障则会导致中继连接断开，使得中继连接功能暂时性停滞。

![img](/images/decen.png)   

为了能够优化这部分的问题，覆盖高负载访问场景，EdgeMesh 新版本考量使用分布式网络连接的思想，通过给予每一个节点能够提供中继功能的结构，使每一个节点都具有为其他节点提供中继的能力。

针对这部分场景需求，用户可以在集群初始化时指定多个特定的节点作为默认的中继节点，依据自身情况调节集群内负载的分配，EdgeMesh 将会在提供中继服务的时候，优先尝试连接这些节点；如果不做设置，EdgeMesh 也会寻找合适的节点执行中继功能，分散减轻单个节点的中继访问负担。



### 分布式动态中继连接场景

如图所示，位于上海的边缘应用A和B通过中继互相通信，需要把流量转发到处于北京数据中心里的 relay 节点，数据传输在远距离的两地之间绕了一圈，导致服务时延较长，用户体验较差。非常遗憾的是，边缘计算场景下集群规模经常横跨多地或者是多区域部署，如果中继节点距离请求服务的节点非常遥远，就会造成极大的延迟，影响用户的体验，这个情况尤其是在中继连接对象与自己在相邻地理位置下的时候，体现得尤为明显。

![img](/images/farButnear.png)

为了能够优化这部分的体验，覆盖远距离服务场景，EdgeMesh 新版本考量就近中继的原则，用户可以根据集群节点的地理位置分布情况，支持选择一个地理位置适中的 relay 节点。当应用需要中继连接服务的时候，edgemesh-agent 就会动态优先选择就近的 relay 节点作为中继来提供网络连接服务，以此缩短中继服务的时延。



### 私有局域网网络自治场景

如图所示，在老版本的 EdgeMesh 的代码实现中，edgemesh-agent 必须保持与云上中继服务edgemesh-server 的连接，当局域网内的节点离线后，导致 edgemesh-agent 断开与中继节点的连接，断连节点上的服务就彻底失去流量代理的能力了，这在部分私有局域网网络内或者是网络情况波动较大的环境当中会给用户造成较大的困扰。

![img](/images/mDNS.png)

为了能够优化这部分的问题，提高网络应用连接的稳定性，EdgeMesh 新版本考量了分布式管理及网络自治的想法，让 EdgeMesh 能够通过 mDNS 机制保障私有局域网网络内或者是离线局域网内节点之间的相互发现和转发流量，维持应用服务的正常运转。

针对这部分场景需求，用户并不需要再单独设置任何的参数来启用此功能，该功能一般面对两种情形进行服务维持：

1. 在刚部署 EdgeMesh 的时候，部分节点就已经在私有局域网下，那这个局域网内的节点依旧可以通过 EdgeMesh 来相互之间访问和转发流量。
2. 在集群正常运转过程当中，部分节点离线后，这部分节点依旧可以通过 EdgeMesh 来维持相互之间的网络连接和流量转发。