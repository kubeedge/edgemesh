# 介绍

EdgeMesh 作为 [KubeEdge](https://github.com/kubeedge/kubeedge) 集群的数据面组件，为应用程序提供了简单的服务发现与流量代理功能，从而屏蔽了边缘场景下复杂的网络结构。

## 背景

KubeEdge 基于 [Kubernetes](https://github.com/kubernetes/kubernetes) 构建，将云原生容器化应用程序编排能力延伸到了边缘。但是，在边缘计算场景下，网络拓扑较为复杂，不同区域中的边缘节点往往网络不互通，并且应用之间流量的互通是业务的首要需求，而 EdgeMesh 正是对此提供了一套解决方案。

## 优势

EdgeMesh 满足边缘场景下的新需求（如边缘资源有限、边云网络不稳定、网络结构复杂等），即实现了高可用性、高可靠性和极致轻量化：

- **高可用性**
  - 利用 LibP2P 提供的能力，来打通边缘节点间的网络
  - 将边缘节点间的通信分为局域网内和跨局域网
    - 局域网内的通信：直接通信
    - 跨局域网的通信：打洞成功时 Agent 之间建立直连通道，否则通过 Server 中继转发
- **高可靠性 （离线场景）**
  - 元数据通过 KubeEdge 边云通道下发，无需访问云端 apiserver
  - EdgeMesh 内部集成轻量的节点级 DNS 服务器，服务发现不依赖云端 CoreDNS
- **极致轻量化**
  - 每个节点有且仅有一个 Agent，节省边缘资源

**用户价值**

- 使用户具备了跨越不同局域网边到边/边到云/云到边的应用互访能力
- 相对于部署 CoreDNS + Kube-Proxy + CNI 这一套组件，用户只需要在节点部署一个 Agent 就能完成目标

## 关键功能

<table align="center">
  <tr>
    <th align="center">功能</th>
    <th align="center">子功能</th>
    <th align="center">实现度</th>
  </tr>
  <tr>
    <td align="center">服务发现</td>
    <td align="center">/</td>
    <td align="center">✓</td>
  </tr>
  <tr>
    <td rowspan="5" align="center">流量治理</td>
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
    <td align="center">UDP</td>
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
    <td rowspan="2" align="center">边缘网关</td>
    <td align="center">外部访问</td>
    <td align="center">✓</td>
  </tr>
  <tr>
    <td align="center">多网卡监听</td>
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
