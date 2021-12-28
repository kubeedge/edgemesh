# Configuration

## Helm Configuration

### edgemesh

#### 1. Agent Subchart

The agent subchart configuration declaration is in build/helm/edgemesh/charts/agent/values.yaml.

#### 1.1 image

meaning: specify the image used by edgemesh-agent

example: --set agent.image=kubeedge/edgemesh-agent:v1.9.0

#### 1.2 configmap configuration

meaning: all parameters in [ConfigMap Configuration](#_1-edgemesh-agent-1) of edgemesh-agent below can be referenced by the agent

example: --set agent.modules.edgeProxy.socks5Proxy.enable=true

#### 2. Server Subchart

The server subchart configuration declaration is in build/helm/edgemesh/charts/server/values.yaml.

#### 2.1 image

meaning: specify the image used by edgemesh-server

example: --set server.image=kubeedge/edgemesh-server:v1.9.0

#### 2.2 nodeName

meaning: specify the scheduled worker node of edgemesh-server

example: --set server.nodeName=k8s-node1

#### 2.3 advertiseAddress

meaning: specify the list of service IPs exposed by edgemesh-server, separate multiple IPs with commas

example: --set "server.advertiseAddress={119.8.211.54,100.10.1.4}"

#### 2.4 configmap configuration

meaning: all the parameters in the [ConfigMap Configuration](#_2-edgemesh-server-1) of the edgemesh-server below can be referenced by the server

example: --set server.modules.tunnel.listenPort=20005

### edgemesh-gateway

The edgemesh-gateway chart configuration declaration is in build/helm/gateway/values.yaml.

#### 3.1 image

meaning: specify the image used by edgemesh-agent

example: --set image=kubeedge/edgemesh-agent:v1.9.0

#### 3.2 nodeName

meaning: specify the scheduled worker node of edgemesh-gateway

example: --set nodeName=ke-edge1

#### 3.3 configmap configuration

meaning: all parameters in [ConfigMap Configuration](#_1-edgemesh-agent-1) of edgemesh-agent below can be referenced

example: --set modules.tunnel.listenPort=20009

## ConfigMap Configuration

### edgemesh-agent

#### configuration example:

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

#### Table 1: edgemesh-agent

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| apiVersion | string | agent.edgemesh.config.kubeedge.io/v1alpha1 | API version |
| kind | string | EdgeMeshAgent | API Type |
| kubeAPIConfig | object | [Table 1-1](#t1-1) | Kubernetes API configuration |
| commonConfig | object | [Table 1-2](#t1-2) | edgemesh-agent common configuration |
| goChassisConfig | object | [Table 1-3](#t1-3) | go chassis configuration |
| modules | object | [Table 1-4](#t1-4) | contains all sub-modules of edgemesh-agent |

<a name="t1-1"></a>

#### Table 1-1: kubeAPIConfig

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| master | string | empty | Kubernetes API server address. If it is not filled in, the program will automatically determine: "empty" on the cloud, and "127.0.0.1:10550" on the edge |
| contentType | string | empty | Type of message transmission during Kubernetes interaction. If it is not filled in, the program will automatically determine: "application/vnd.kubernetes.protobuf" on the cloud, and "application/json" on the edge |
| qps | int32 | 100 | qps when talking with Kubernetes apiserver |
| burst | int32 | 200 | burst when talking with Kubernetes apiserve |
| kubeConfig | string | empty | kubeconfig file path |

<a name="t1-2"></a>

#### Table 1-2: commonConfig

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| mode | string | DebugMode | The operating mode (DebugMode, CloudMode, EdgeMode) that edgemesh-agent is in. No manual configuration required, automatically recognized by the program |
| configMapName | string | edgemesh-agent-cfg | name of configmap mounted by edgemesh-agent |
| dummyDeviceName | string | edgemesh0 | Name of the network card created by edgemesh-agent |
| dummyDeviceIP | string | 169.254.96.16 | IP of the network card created by edgemesh-agent |

<a name="t1-3"></a>

#### Table 1-3: goChassisConfig

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| loadBalancer | object | [Table 1-3-1](#t1-3-1) | load balancing configuration |
| protocol | object | [Table 1-3-2](#t1-3-2) | communication protocol configuration |

<a name="t1-3-1"></a>

#### Table 1-3-1: loadBalancer

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| defaultLBStrategy | string | RoundRobin | default load balancing strategy |
| supportLBStrategies | []string | [RoundRobin, Random, ConsistentHash] | list of supported load balancing strategies  |
| consistentHash | object | [Table 1-3-1-1](#t1-3-1-1) | consistent hash strategy |

<a name="t1-3-1-1"></a>

#### Table 1-3-1-1: consistentHash

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| load | float64 | 1.25 | hash ring bounded load value |
| partitionCount | int | 100 | number of hash ring partitions |
| replicationFactor | int | 10 | hash ring repetition factor |

<a name="t1-3-2"></a>

#### Table 1-3-2: protocol

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| tcpBufferSize | int | 8192 | buffer size for TCP communication |
| tcpClientTimeout | int | 5 | TCP client timeout time, in seconds |
| tcpReconnectTimes | int | 3 | number of retries for TCP connection establishment |
| nodeName | string | empty | the name of the node where edgemesh-agent is scheduled. No manual configuration required, automatically recognized by the program |

<a name="t1-4"></a>

#### Table 1-4: modules

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| edgeDNS | object | [Table 1-4-1](#t1-4-1) | edgedns submodule, a built-in lightweight DNS server |
| edgeProxy | object | [Table 1-4-2](#t1-4-2) | edgeproxy submodule, a proxy server of various protocols |
| edgeGateway | object | [Table 1-4-3](#t1-4-3) | edgegateway submodule, which provides an ingress gateway for external access |
| tunnel | object | [Table 1-4-4](#t1-4-4) | tunnelagent submodule, which uses relay and hole punching to provide the ability to communicate across subnets |

<a name="t1-4-1"></a>

#### Table 1-4-1: edgeDNS

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| enable | bool | empty | submodule start switch. If it is not filled in, the program will automatically determine: "false" on the cloud, and "true" on the edge |
| listenPort | int | 53 | the port that the DNS server listens on |

<a name="t1-4-2"></a>

#### Table 1-4-2: edgeProxy

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| enable | bool | false | submodule start switch |
| listenPort | int | 53 | TCP proxy listening port |
| subNet | string | empty | The Cluster IP network segment of the Kubernetes cluster. No manual configuration required, automatically recognized by the program |
| socks5Proxy | object | [Table 1-4-2-1](#t1-4-2-1) | socks5 proxy submodule |

<a name="t1-4-2-1"></a>

#### Table 1-4-2-1: socks5Proxy

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| enable | bool | false | submodule start switch |
| listenPort | int | 10080 | socks5 proxy listening port |

<a name="t1-4-3"></a>

#### Table 1-4-3: edgeGateway

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| enable | bool | false | submodule start switch |
| nic | string | * | the list of network cards that the edge gateway needs to listen, such as "lo,eth0"; empty or "*" means to monitor all network cards |
| includeIP | string | * | the IP list that the edge gateway needs to listen, such as "192.168.1.56,10.3.2.1"; blank or "*" means to monitor all network cards |
| excludeIP | string | * | the IP list that needs to be filtered by the edge gateway, such as "192.168.1.56,10.3.2.1"; empty or "*" instead of Table No IP needs to be filtered |

<a name="t1-4-4"></a>

#### Table 1-4-4: tunnel

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| enable | bool | false | submodule start switch |
| listenPort | int | 53 | the port that tunnelagent listens to |
| nodeName | string | empty | the name of the node where edgemesh-agent is scheduled. No manual configuration required, automatically recognized by the program |
| security | object | [Table 1-4-4-1](#t1-4-4-1) | tunnel security configuration |
| enableHolePunch | bool | true | p2p hole punching option |

<a name="t1-4-4-1"></a>

#### Table 1-4-4-1: security

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| enable | bool | false | submodule start switch |
| tlsCaFile | string | /etc/kubeedge/edgemesh/agent/acls/rootCA.crt | CA file path |
| tlsCertFile | string | /etc/kubeedge/edgemesh/agent/acls/server.crt | certificate file path |
| tlsPrivateKeyFile | string | /etc/kubeedge/edgemesh/agent/acls/server.key | private key file path |
| token | string | empty | token. No manual configuration required, auto fetch from  `tokensecret` secret in kubeedge namespace |
| httpServer | string | empty | the address used to download the certificate, which is equivalent to the advertiseAddress of cloudcore |

::: tip
edgemesh-agent and edgemesh-gateway use the same configmap configuration, no additional description here.
:::

### edgemesh-server

#### configuration example:

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

#### Table 2: edgemesh-server

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| apiVersion | string | server.edgemesh.config.kubeedge.io/v1alpha1 | API version |
| kind | string | EdgeMeshServer | API type |
| kubeAPIConfig | object | [Table 1-1](#t1-1) | Kubernetes API configuration |
| modules | object | [Table 2-1](#t2-1) | contains all submodules of edgemesh-server |

#### Table 2-1: modules

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| tunnel | object | [Table 2-1-1](#t2-1-1) | tunnelserver submodule |

<a name="t2-1-1"></a>

#### Table 2-1-1: tunnel

| Name | Type | Default | Description |
| ---- | ---- | ---- | ---- |
| enable | bool | false | submodule start switch |
| listenPort | int | 20004 | the port that tunnelserver listens to |
| advertiseAddress | []string | empty | IP list of services exposed by edgemesh-server |
| nodeName | string | empty | the name of the node where edgemesh-server is scheduled. No manual configuration required, automatically recognized by the program |
| security | object | [Table 1-4-4-1](#t1-4-4-1) | tunnel security configuration |
