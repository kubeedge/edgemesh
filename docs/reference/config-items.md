# Configure

## Helm configure

### EdgeMesh

The Helm Chart configuration of EdgeMesh is placed in the build/helm/edgemesh directory.

#### 1. edgemesh-agent

| Name       | Type   | Example of use                                        | Describe                                   |
|------------|--------|-------------------------------------------------------|--------------------------------------------|
| image      | string | --set agent.image=kubeedge/edgemesh-agent:v1.12.0     | Specifies the image used by edgemesh-agent |
| psk        | string | --set agent.psk=123456                                | PSK cipher                                 |
| relayNodes | list   | --set relayNodes[0].nodeName=k8s-master               | Relay node configuration table             |
| modules    | object | --set agent.modules.edgeProxy.socks5Proxy.enable=true | Same meaning as modules in configmap       |

### Edgemesh-Gateway

The Helm Chart configuration of EdgeMesh-Gateway is placed in the build/helm/edgemesh-gateway directory.

| Name       | Type   | Example of use                                | Describe                                            |
|------------|--------|-----------------------------------------------|-----------------------------------------------------|
| image      | string | --set image=kubeedge/edgemesh-gateway:v1.12.0 | Specifies the image used by edgemesh-gateway        |
| nodeName   | string | --set nodeName=k8s-master                     | Specify the node where edgemesh-gateway is deployed |
| psk        | string | --set psk=123456                              | PSK cipher                                          |
| relayNodes | list   | --set relayNodes[0].nodeName=k8s-master       | Relay node configuration table                      |
| modules    | object | --set modules.edgeGateway.nic=eth0            | Same meaning as modules in configmap                |

## ConfigMap Configure

:::tip
For detailed field explanation, please refer to [API Definition](https://github.com/kubeedge/edgemesh/blob/main/pkg/apis/config/v1alpha1/type.go).
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
  metaServerAddress: http://127.0.0.1:10550
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
  metaServerAddress: http://127.0.0.1:10550
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
```
