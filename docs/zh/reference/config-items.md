# 配置

## Helm 配置

### EdgeMesh

EdgeMesh 的 Helm Chart 配置放在 build/helm/edgemesh 目录下。

#### 1. edgemesh-agent

| 名称               | 类型     | 使用示例                                                  | 描述                               |
|------------------|--------|-------------------------------------------------------|----------------------------------|
| image            | string | --set agent.image=kubeedge/edgemesh-agent:v1.12.0     | 指定 edgemesh-agent 使用的镜像          |
| psk              | string | --set agent.psk=123456                                | PSK 密码                           |
| relayNodes       | list   | --set agent.relayNodes[0].nodeName=k8s-master         | 中继节点配置表                          |
| metaServerSecret | string | --set agent.metaServerSecret=metaserver-certs         | 存放 metaServer 证书文件的 Secret       |
| kubeAPIConfig    | object | --set agent.kubeAPIConfig.master=https://1.1.1.1:6443 | 与 configmap 的 kubeAPIConfig 含义相同 |
| commonConfig     | object | --set agent.commonConfig.bridgeDeviceIP=169.254.96.16 | 与 configmap 的 commonConfig 含义相同  |
| modules          | object | --set agent.modules.edgeProxy.socks5Proxy.enable=true | 与 configmap 的 modules 含义相同       |

### EdgeMesh-Gateway

EdgeMesh-Gateway 的 Helm Chart 配置放在 build/helm/edgemesh-gateway 目录下。

| 名称               | 类型     | 使用示例                                            | 描述                               |
|------------------|--------|-------------------------------------------------|----------------------------------|
| image            | string | --set image=kubeedge/edgemesh-gateway:v1.12.0   | 指定 edgemesh-gateway 使用的镜像        |
| nodeName         | string | --set nodeName=k8s-master                       | 指定 edgemesh-gateway 部署的节点        |
| psk              | string | --set psk=123456                                | PSK 密码                           |
| relayNodes       | list   | --set relayNodes[0].nodeName=k8s-master         | 中继节点配置表                          |
| metaServerSecret | string | --set metaServerSecret=metaserver-certs         | 存放 metaServer 证书文件的 Secret       |
| kubeAPIConfig    | object | --set kubeAPIConfig.master=https://1.1.1.1:6443 | 与 configmap 的 kubeAPIConfig 含义相同 |
| modules          | object | --set modules.edgeGateway.nic=eth0              | 与 configmap 的 modules 含义相同       |

## ConfigMap 配置

:::tip
详细的字段解释，请参考 [API定义](https://github.com/kubeedge/edgemesh/blob/main/pkg/apis/config/v1alpha1/types.go)。
:::

### edgemesh-agent-cfg

```yaml
apiVersion: agent.edgemesh.config.kubeedge.io/v1alpha1
kind: EdgeMeshAgent
kubeAPIConfig:
  master: https://119.8.211.54:6443
  contentType: application/vnd.kubernetes.protobuf
  qps: 100
  burst: 200
  kubeConfig: /root/.kube/config
  metaServer:
    server: http://127.0.0.1:10550
    security:
      requireAuthorization: false
      insecureSkipTLSVerify: false
      tlsCaFile: /etc/edgemesh/metaserver/rootCA.crt
      tlsCertFile: /etc/edgemesh/metaserver/server.crt
      tlsPrivateKeyFile: /etc/edgemesh/metaserver/server.key
  deleteKubeConfig: false
commonConfig:
  bridgeDeviceName: edgemesh0
  bridgeDeviceIP: 169.254.96.16
modules:
  edgeDNS:
    enable: false
    listenPort: 53
    cacheDNS:
      enable: false
      autoDetect: true
      upstreamServers:
      - 10.96.0.10
      - 1.1.1.1
      cacheTTL: 30
  edgeProxy:
    enable: false
    socks5Proxy:
      enable: false
      listenPort: 10800
    loadBalancer:
      consistentHash:
        partitionCount: 100
        replicationFactor: 10
        load: 1.25
    serviceFilterMode: FilterIfLabelExists
  edgeTunnel:
    enable: false
    listenPort: 20006
    transport: tcp
    rendezvous: EDGEMESH_PLAYGOUND
    relayNodes:
    - nodeName: k8s-master
      advertiseAddress:
      - 1.1.1.1
    - nodeName: ke-edge1
      advertiseAddress:
      - 2.2.2.2
      - 3.3.3.3
    enableIpfsLog: false
    maxCandidates: 5
    heartbeatPeriod: 120
    finderPeriod: 60
    psk:
      enable: true
      path: /etc/edgemesh/psk
    tunnelLimitConfig:
      enable: false
      tunnelBaseStreamIn: 10240
      tunnelBaseStreamOut: 10240
      TunnelPeerBaseStreamIn: 1024
      tunnelPeerBaseStreamOut: 1024
```

### edgemesh-gateway-cfg

```yaml
apiVersion: gateway.edgemesh.config.kubeedge.io/v1alpha1
kind: EdgeMeshGateway
kubeAPIConfig:
  master: https://119.8.211.54:6443
  contentType: application/vnd.kubernetes.protobuf
  qps: 100
  burst: 200
  kubeConfig: /root/.kube/config
  metaServer:
    server: http://127.0.0.1:10550
    security:
      requireAuthorization: false
      insecureSkipTLSVerify: false
      tlsCaFile: /etc/edgemesh/metaserver/rootCA.crt
      tlsCertFile: /etc/edgemesh/metaserver/server.crt
      tlsPrivateKeyFile: /etc/edgemesh/metaserver/server.key
  deleteKubeConfig: false
modules:
  edgeGateway:
    enable: false
    nic: "lo,eth0",
    includeIP: "192.168.0.1,172.16.0.1",
    excludeIP: "10.0.0.1",
    loadBalancer:
      consistentHash:
        partitionCount: 100
        replicationFactor: 10
        load: 1.25
  edgeTunnel:
    enable: false
    listenPort: 20006
    transport: tcp
    rendezvous: EDGEMESH_PLAYGOUND
    relayNodes:
    - nodeName: k8s-master
      advertiseAddress:
      - 1.1.1.1
    - nodeName: ke-edge1
      advertiseAddress:
      - 2.2.2.2
      - 3.3.3.3
    enableIpfsLog: false
    maxCandidates: 5
    heartbeatPeriod: 120
    finderPeriod: 60
    psk:
      enable: true
      path: /etc/edgemesh/psk
    tunnelLimitConfig:
      enable: false
      tunnelBaseStreamIn: 10240
      tunnelBaseStreamOut: 10240
      TunnelPeerBaseStreamIn: 1024
      tunnelPeerBaseStreamOut: 1024
```
