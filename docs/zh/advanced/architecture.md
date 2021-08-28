# 架构

## 概览

![edgemesh-architecture](/images/advanced/em-arch.png)

上图展示了 EdgeMesh 的简要架构，EdgeMesh 包含两个微服务：EdgeMesh-Server 和 EdgeMesh-Agent。

EdgeMesh-Server 的核心组件包括：

- **Tunnel-Server**: 基于 [LibP2P](https://github.com/libp2p/go-libp2p) 实现，与 EdgeMesh-Agent 建立连接，为EdgeMesh-Agent 提供中继能力和打洞能力

EdgeMesh-Agent 的核心组件包括：

- **Proxier**: 负责配置内核的 iptables 规则，将请求拦截到 EdgeMesh 进程内
- **DNS**: 内置的 DNS 解析器，将节点内的域名请求解析成一个服务的集群 IP
- **Traffic**: 基于 Go-Chassis 框架的流量转发模块，负责转发应用间的流量
- **Controller**: 通过 KubeEdge 的边缘侧 List-Watch 能力获取 Service、Endpoints、Pod 等元数据
- **Tunnel-Agent**: 基于 LibP2P 实现，利用中继和打洞来提供跨子网通讯的能力

:::tip
为了保证一些低版本内核、低版本 iptables 边缘设备的服务发现能力，EdgeMesh-Agent 在流量代理的实现上采用了 userspace 模式。
:::

## 工作原理

- EdgeMesh-Agent 通过 KubeEdge 边缘侧 List-Watch 的能力，监听 Service、Endpoints 等元数据的增删改，维护访问服务所需要的元数据; 同时配置 iptables 规则拦截 Cluster IP 网段的请求
- EdgeMesh-Agent 使用与 K8s Service 相同的 Cluster IP 和域名的方式来访问服务
- 假设我们有 APP-A 和 APP-B 两个服务，当 APP-A 服务基于域名访问 APP-B 时，域名解析请求会被本节点的 EdgeMesh-Agent 拦截并返回 Cluster IP，这个请求会被 EdgeMesh-Agent 之前配置的 iptables 规则重定向，转发到 EdgeMesh-Agent 进程的 40001 端口里（数据包从内核态->用户态）
- 请求进入 EdgeMesh-Agent 进程后，由 EdgeMesh-Agent 进程完成后端 Pod 的选择（负载均衡在这里发生），然后这个请求会通过 tunnel 模块发到 APP-B 所在主机的 EdgeMesh-Agent 上（通过中继转发或者打洞直接传输）
- App-B 所在节点的 EdgeMesh-Agent 负责将流量转发到 APP-B 的服务端口上，并获取响应返回给 APP-A 所在节点的 EdgeMesh-Agent
- APP-A 所在节点的 EdgeMesh-Agent 负责将响应数据反馈给 APP-A 服务
