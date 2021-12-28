# 配置

## Helm 配置

### edgemesh

#### 1. agent 子Chart

agent 子Chart 配置声明在 build/helm/edgemesh/charts/agent/values.yaml 中。

#### 1.1 image

含义：指定 edgemesh-agent 使用的镜像

示例：--set agent.image=kubeedge/edgemesh-agent:v1.9.0

#### 1.2 configmap 配置

含义：下方 edgemesh-agent 的 [configmap 配置](#_1-edgemesh-agent-1) 中所有参数都能被 agent 引用

示例：--set agent.modules.edgeProxy.socks5Proxy.enable=true

#### 2. server 子Chart

server 子Chart 配置声明在 build/helm/edgemesh/charts/server/values.yaml 中。

#### 2.1 image

含义：指定 edgemesh-server 使用的镜像

示例：--set server.image=kubeedge/edgemesh-server:v1.9.0

#### 2.2 nodeName

含义：指定 edgemesh-server 被调度的工作节点

示例：--set server.nodeName=k8s-node1

#### 2.3 advertiseAddress

含义：指定 edgemesh-server 对外暴露的服务 IP 列表，多个 IP 之间使用逗号分隔

示例：--set "server.advertiseAddress={119.8.211.54,100.10.1.4}"

#### 2.4 configmap 配置

含义：下方 edgemesh-server 的 [configmap 配置](#_2-edgemesh-server-1) 中所有参数都能被 server 引用

示例：--set server.modules.tunnel.listenPort=20005

### edgemesh-gateway

edgemesh-gateway 的 chart 配置声明在 build/helm/gateway/values.yaml 中。

#### 3.1 image

含义：指定 edgemesh-gateway 使用的镜像

示例：--set image=kubeedge/edgemesh-agent:v1.9.0

#### 3.2 nodeName

含义：指定 edgemesh-gateway 被调度的工作节点

示例：--set nodeName=ke-edge1

#### 3.3 configmap 配置

含义：下方 edgemesh-agent 的 [configmap 配置](#_1-edgemesh-agent-1) 中所有参数都能被引用

示例：--set modules.tunnel.listenPort=20009

## ConfigMap 配置

### edgemesh-agent

#### 配置示例：

```shell
apiVersion: agent.edgemesh.config.kubeedge.io/v1alpha1
kind: EdgeMeshAgent
kubeAPIConfig:
  master: https://119.8.211.54:6443
  kubeConfig: /root/.kube/config
  qps: 100
  burst: 200
commonConfig:
  mode: DebugMode
  configMapName: edgemesh-agent-cfg
  dummyDeviceIP: 169.254.96.16
  dummyDeviceName: edgemesh0
goChassisConfig:
  loadBalancer:
    consistentHash:
      load: 1.25
      partitionCount: 100
      replicationFactor: 10
    defaultLBStrategy: RoundRobin
    supportLBStrategies:
    - RoundRobin
    - Random
    - ConsistentHash
  protocol:
    tcpBufferSize: 8192
    tcpClientTimeout: 5
    tcpReconnectTimes: 3
    nodeName: ke-edge1
modules:
  edgeDNS:
    enable: true
    listenPort: 53
  edgeProxy:
    enable: true
    listenPort: 40001
    socks5Proxy:
      enable: true
      listenPort: 10800
    subNet: 10.96.0.0/12
  edgeGateway:
    enable: true
    nic: *
    includeIP: *
    excludeIP: *
  tunnel:
    enable: true
    listenPort: 20006
    nodeName: ke-edge1
    security:
      enable: true
      tlsCaFile: /etc/kubeedge/edgemesh/agent/acls/rootCA.crt
      tlsCertFile: /etc/kubeedge/edgemesh/agent/acls/server.crt
      tlsPrivateKeyFile: /etc/kubeedge/edgemesh/agent/acls/server.key
      token: "4cdfc5b7dc37b716feb0156eebaad1c00297145088c9d069a797a94d2379410b.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzQ3MTI0Mjd9.w2l5pulUdI1IekLe-ZngvATjlhhqwsvrTjv_aNSBjF8"
      httpServer: https://119.8.211.54:10002
```

#### 表1: edgemesh-agent

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| apiVersion | string | agent.edgemesh.config.kubeedge.io/v1alpha1 | API 版本号 |
| kind | string | EdgeMeshAgent | API 类型 |
| kubeAPIConfig | object | [表1-1](#t1-1) | Kubernetes API 配置 |
| commonConfig | object | [表1-2](#t1-2) | edgemesh-agent 公共配置项 |
| goChassisConfig | object | [表1-3](#t1-3) | go chassis 配置项 |
| modules | object | [表1-4](#t1-4) | 包含 edgemesh-agent 所有子模块 |

<a name="t1-1"></a>

#### 表1-1: kubeAPIConfig

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| master | string | 空 | Kubernetes API 服务器地址。不填写则由程序自动判断：云上为"空"，边上为"127.0.0.1:10550" |
| contentType | string | 空 | Kubernetes 交互时消息传输的类型。不填写则由程序自动判断：云上为"application/vnd.kubernetes.protobuf"，边上为"application/json" |
| qps | int32 | 100 | 与 kubernetes apiserve 交谈时的 qps |
| burst | int32 | 200 | 与 kubernetes apiserve 交谈时的 burst |
| kubeConfig | string | 空 | kubeconfig 文件路径 |

<a name="t1-2"></a>

#### 表1-2: commonConfig

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| mode | string | DebugMode | edgemesh-agent 处于的运行模式(DebugMode, CloudMode, EdgeMode)。无需手动配置，由程序自动识别 |
| configMapName | string | edgemesh-agent-cfg | edgemesh-agent 挂载的 configmap 的名称 |
| dummyDeviceName | string | edgemesh0 | edgemesh-agent 创建的网卡名称 |
| dummyDeviceIP | string | 169.254.96.16 | edgemesh-agent 创建的网卡 IP |

<a name="t1-3"></a>

#### 表1-3: goChassisConfig

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| loadBalancer | object | [表1-3-1](#t1-3-1) | 负载均衡配置 |
| protocol | object | [表1-3-2](#t1-3-2) | 通信协议配置 |

<a name="t1-3-1"></a>

#### 表1-3-1: loadBalancer

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| defaultLBStrategy | string | RoundRobin | 默认的负载均衡策略 |
| supportLBStrategies | []string | [RoundRobin, Random, ConsistentHash] | 支持的负载均衡策略列表 |
| consistentHash | object | [表1-3-1-1](#t1-3-1-1) | 一致性哈希策略 |

<a name="t1-3-1-1"></a>

#### 表1-3-1-1: consistentHash

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| load | float64 | 1.25 | 哈希环有界负载值 |
| partitionCount | int | 100 | 哈希环分区数 |
| replicationFactor | int | 10 | 哈希环重复因子 |

<a name="t1-3-2"></a>

#### 表1-3-2: protocol

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| tcpBufferSize | int | 8192 | TCP 通信的缓存区大小 |
| tcpClientTimeout | int | 5 | TCP 客户端超时时间，单位秒 |
| tcpReconnectTimes | int | 3 | TCP 建立连接的重试次数 |
| nodeName | string | 无 | edgemesh-agent 被调度到的节点名称。无需手动配置，由程序自动识别 |

<a name="t1-4"></a>

#### 表1-4: modules

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| edgeDNS | object | [表1-4-1](#t1-4-1) | edgedns 子模块，内置的轻量级 DNS 服务器 |
| edgeProxy | object | [表1-4-2](#t1-4-2) | edgeproxy 子模块，各种协议的代理服务器 |
| edgeGateway | object | [表1-4-3](#t1-4-3) | edgegateway 子模块，提供外部访问的入口网关 |
| tunnel | object | [表1-4-4](#t1-4-4) | tunnelagent 子模块，利用中继和打洞来提供跨子网通讯的能力 |

<a name="t1-4-1"></a>

#### 表1-4-1: edgeDNS

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| enable | bool | 空 | 子模块启动开关。不填写则由程序自动判断：云上为"false"，边上为"true" |
| listenPort | int | 53 | DNS 服务器监听的端口 |

<a name="t1-4-2"></a>

#### 表1-4-2: edgeProxy

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| enable | bool | false | 子模块启动开关 |
| listenPort | int | 53 | TCP 代理监听的端口 |
| subNet | string | 无 | Kubernetes 集群的 Cluster IP 网段。无需手动配置，由程序自动识别 |
| socks5Proxy | object | [表1-4-2-1](#t1-4-2-1) | socks5 代理子模块 |

<a name="t1-4-2-1"></a>

#### 表1-4-2-1: socks5Proxy

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| enable | bool | false | 子模块启动开关 |
| listenPort | int | 10080 | Socks5 代理监听的端口 |

<a name="t1-4-3"></a>

#### 表1-4-3: edgeGateway

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| enable | bool | false | 子模块启动开关 |
| nic | string | * | 边缘网关需要监听的网卡列表，例如"lo,eth0"；空或"*"代表监听所有网卡 |
| includeIP | string | * | 边缘网关需要监听的 IP 列表，例如"192.168.1.56,10.3.2.1"；空或"*"代表监听所有网卡 |
| excludeIP | string | * | 边缘网关需要过滤的 IP 列表，例如"192.168.1.56,10.3.2.1"；空或"*"代表无 IP 需要过滤 |

<a name="t1-4-4"></a>

#### 表1-4-4: tunnel

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| enable | bool | false | 子模块启动开关 |
| listenPort | int | 53 | tunnelagent 监听的端口 |
| nodeName | string | 无 | edgemesh-agent 被调度到的节点名称。无需手动配置，由程序自动识别 |
| security | object | [表1-4-4-1](#t1-4-4-1) | tunnel security 配置项 |
| enableHolePunch | bool | true | p2p 打洞开关 |

<a name="t1-4-4-1"></a>

#### 表1-4-4-1: security

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| enable | bool | false | 子模块启动开关 |
| tlsCaFile | string | /etc/kubeedge/edgemesh/agent/acls/rootCA.crt | CA 文件路径 |
| tlsCertFile | string | /etc/kubeedge/edgemesh/agent/acls/server.crt | 证书文件路径 |
| tlsPrivateKeyFile | string | /etc/kubeedge/edgemesh/agent/acls/server.key | 私钥文件路径 |
| token | string | 无 | 口令。无需手动配置，这个值会通过挂载 kubeedge namespace 下面的 `tokensecret` secret 自动获取 |
| httpServer | string | 无 | 用于下载证书的地址。等同于 cloudcore 的 advertiseAddress |

::: tip
edgemesh-agent 与 edgemesh-gateway 使用了相同的 configmap 配置，此处不再额外描述。
:::

### edgemesh-server

#### 配置示例：

```shell
apiVersion: server.edgemesh.config.kubeedge.io/v1alpha1
kind: EdgeMeshServer
kubeAPIConfig:
  master: https://119.8.211.54:6443
  kubeConfig: /root/.kube/config
  qps: 100
  burst: 200
modules:
  tunnel:
    enable: true
    listenPort: 20004
    nodeName: k8s-master
    security:
      enable: true
      tlsCaFile: /etc/kubeedge/edgemesh/server/acls/rootCA.crt
      tlsCertFile: /etc/kubeedge/edgemesh/server/acls/server.crt
      tlsPrivateKeyFile: /etc/kubeedge/edgemesh/server/acls/server.key
      token: "4cdfc5b7dc37b716feb0156eebaad1c00297145088c9d069a797a94d2379410b.eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzQ3MTI0Mjd9.w2l5pulUdI1IekLe-ZngvATjlhhqwsvrTjv_aNSBjF8"
      httpServer: https://119.8.211.54:10002
```

#### 表2: edgemesh-server

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| apiVersion | string | server.edgemesh.config.kubeedge.io/v1alpha1 | API 版本号 |
| kind | string | EdgeMeshServer | API 类型 |
| kubeAPIConfig | object | [表1-1](#t1-1) | Kubernetes API 配置 |
| modules | object | [表2-1](#t2-1) | 包含 edgemesh-server 所有子模块 |

#### 表2-1: modules

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| tunnel | object | [表2-1-1](#t2-1-1) | tunnelserver 子模块 |

<a name="t2-1-1"></a>

#### 表2-1-1: tunnel

| 名称 | 类型 | 默认值 | 描述 |
| ---- | ---- | ---- | ---- |
| enable | bool | false | 子模块启动开关 |
| listenPort | int | 20004 | tunnelserver 监听的端口 |
| advertiseAddress | []string | 无 | edgemesh-server 对外暴露的服务 IP 列表 |
| nodeName | string | 无 | edgemesh-server 被调度到的节点名称。无需手动配置，由程序自动识别 |
| security | object | [表1-4-4-1](#t1-4-4-1) | tunnel security 配置项 |
