# 边缘 Kube-API 端点

## 背景

Kubernetes 通过 CRD 和 Controller 机制极大程度的提升了自身的可扩展性，使得众多应用能轻松的集成至 Kubernetes 生态。众所周知，大部分 Kubernetes 应用会通过访问 kube-apiserver 获取基本的元数据，比如 Service、Pod、Job 和 Deployment 等等，以及获取基于自身业务扩展的 CRD 的元数据。

然而，在边缘计算场景下由于网络不互通，导致边缘节点通常无法直接连接到处于云上的 kube-apiserver 服务，使得部署在边缘的 Kubernetes 应用无法获取它所需要的元数据。比如，被调度到边缘节点的 Kube-Proxy 和 Flannel 通常是无法正常工作的。

**好在，KubeEdge >= v1.7.0 提供了边缘 Kube-API 端点的能力，它能够在边缘提供与云上 kube-apiserver 相似的服务，使得对 kube-apiserver 有需求的边缘应用也能无感知的运行在边缘上。** 本章将指导大家如何配置 KubeEdge 以启用边缘 Kube-API 端点，以及 EdgeMesh 如何与其对接。

## 快速上手

- **步骤1**: 在云端，开启 dynamicController 模块，配置完成后，需要重启 cloudcore

```yaml
$ vim /etc/kubeedge/config/cloudcore.yaml
modules:
  ...
  dynamicController:
    enable: true
...
```

- **步骤2**: 在边缘节点，打开 metaServer 模块（如果你的 KubeEdge < 1.8.0，还需关闭旧版 edgeMesh 模块），配置完成后，需要重启 edgecore

```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edgeMesh:
    enable: false
  ...
  metaManager:
    metaServer:
      enable: true
...
```

- **步骤3**: 在边缘节点，配置 clusterDNS 和 clusterDomain，配置完成后，需要重启 edgecore

```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edged:
    clusterDNS: 169.254.96.16
    clusterDomain: cluster.local
...
```

如果 KubeEdge >= v1.12.0，请这样配置：
```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edged:
    ...
    tailoredKubeletConfig:
      ...
      clusterDNS:
      - 169.254.96.16
      clusterDomain: cluster.local
...
```

::: tip
- 步骤3的配置是为了边缘应用能够访问到 EdgeMesh 的 DNS 服务，与边缘 Kube-API 端点本身无关，但为了配置的流畅性，还是放在这里说明。
- clusterDNS 设置的值 '169.254.96.16' 来自于 [commonConfig](https://edgemesh.netlify.app/zh/reference/config-items.html#edgemesh-agent-cfg) 中 bridgeDeviceIP 的默认值，正常情况下无需修改，非得修改请保持两者一致。
:::

- **步骤4**: 最后，在边缘节点，测试边缘 Kube-API 端点功能是否正常

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

::: warning
如果返回值是空列表，或者响应时长很久（接近 10s）才拿到返回值，说明你的配置可能有误，请仔细检查。
:::

**完成上述步骤之后，KubeEdge 的边缘 Kube-API 端点功能就已经开启了，接着继续部署 EdgeMesh 即可。**

## 安全

KubeEdge >= v1.12.0 对边缘 Kube-API 端点功能进行了 [安全加固](https://github.com/kubeedge/kubeedge/issues/4108)，使其支持 HTTPS 的安全访问。如果你想加固边缘 Kube-API 端点服务的安全性，本节将指导大家如何配置 KubeEdge 以启用安全的边缘 Kube-API 端点功能，以及 EdgeMesh 如何与其对接。

### 配置

- **步骤1**: 开启 KubeEdge 的 requireAuthorization 特性门控

cloudcore.yaml 和 edgecore.yaml 都进行下述配置，配置完成后，需要重启 cloudcore 和 edgecore

```yaml
$ vim /etc/kubeedge/config/cloudcore.yaml
kind: CloudCore
featureGates:
  requireAuthorization: true
modules:
  ...
```

```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
kind: EdgeCore
featureGates:
  requireAuthorization: true
modules:
  ...
```

- **步骤2**: 生成自签名证书

我们借用 KubeEdge 的 `certgen.sh` 脚本来生成临时测试证书。**请注意，请不要在生产环境这样做，生产环境请使用生产级别用的证书。**

```shell
# 1. 确认 /etc/kubernetes/pki/ 目录存在
$ ls /etc/kubernetes/pki/

# 2. 创建目录
$ mkdir -p /tmp/metaserver-certs
$ cd /tmp/metaserver-certs

# 3. 下载 certgen.sh
$ wget https://raw.githubusercontent.com/kubeedge/kubeedge/master/build/tools/certgen.sh
$ chmod u+x certgen.sh

# 4. 生成证书文件
$ CA_PATH=./ CERT_PATH=./ ./certgen.sh stream

# 5. 修改证书名字
$ mv streamCA.crt rootCA.crt; mv stream.crt server.crt; mv stream.key server.key

# 6. 创建证书 secret
$ kubectl -n kubeedge create secret generic metaserver-certs --from-file=./rootCA.crt --from-file=./server.crt --from-file=./server.key
```

- **步骤3**: 在边缘节点，配置 metaServer 的证书路径，配置完成后，需要重启 edgecore

```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  metaManager:
    metaServer:
      enable: true
      server: https://127.0.0.1:10550
      tlsCaFile: /tmp/metaserver-certs/rootCA.crt
      tlsCertFile: /tmp/metaserver-certs/server.crt
      tlsPrivateKeyFile: /tmp/metaserver-certs/server.key
...
```

**完成上述配置并重启后，你就可以拥有一个基于 HTTPS 的、安全的边缘 Kube-API 端点服务，后续你可以参考 [issue#4801](https://github.com/kubeedge/kubeedge/issues/4108) 里的教程，使用 `curl` 测试它能否正常工作。**

### 对接

EdgeMesh 有以下两种方式连接基于 HTTPS 的边缘 Kube-API 端点，你可以根据实际情况二选一使用。

#### 方式一：单向认证

EdgeMesh 可以通过单向认证的方式，访问基于 HTTPS 的边缘 Kube-API 端点，单向认证即不需要验证客户端证书。

- Helm 配置

```shell
helm install edgemesh --namespace kubeedge \
--set agent.kubeAPIConfig.metaServer.security.requireAuthorization=true \
--set agent.kubeAPIConfig.metaServer.security.insecureSkipTLSVerify=true \
...
```

- 手动配置

```yaml
$ vim build/agent/resources/04-configmap.yaml
...
data:
  edgemesh-agent.yaml: |
    kubeAPIConfig:
      metaServer:
        security:
          requireAuthorization: true
          insecureSkipTLSVerify: true
    ...
```

#### 方式二：双向认证

EdgeMesh 也可以通过双向认证的方式，访问基于 HTTPS 的边缘 Kube-API 端点，双向认证需要同时验证服务端证书和客户端证书。

- Helm 配置

```shell
helm install edgemesh --namespace kubeedge \
--set agent.kubeAPIConfig.metaServer.security.requireAuthorization=true \
--set agent.metaServerSecret=metaserver-certs \
...
```

- 手动配置

```yaml
$ vim build/agent/resources/04-configmap.yaml
...
data:
  edgemesh-agent.yaml: |
    kubeAPIConfig:
      metaServer:
        security:
          requireAuthorization: true
    ...
```

```yaml
$ vim build/agent/resources/05-daemonset.yaml
...
        volumeMounts:
        ...
        - name: metaserver-certs
          mountPath: /etc/edgemesh/metaserver
        volumes:
        ...
        - secret:
          secretName: metaserver-certs
          defaultMode: 420
        name: metaserver-certs
        ...
```
