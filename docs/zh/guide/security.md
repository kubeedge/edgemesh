# EdgeMesh 安全配置

EdgeMesh 服务默认没有开启服务认证，edgemesh-server 与 edgemesh-agent 建立连接时没有经过任何认证。如下说明如何通过配置开启 EdgeMesh 服务之间的安全认证，通过如下的配置，EdgeMesh 服务之间会开启身份认证、连接准入、通讯加密。

## 依赖

KubeEdge版本 >= 1.8.2

## 配置

### Helm 配置

支持 Helm 部署时直接开启 security 特性，假设 cloudcore 服务暴露的服务为 110.8.52.21，则部署具体命令如下：

```shell
$ helm install edgemesh --set server.nodeName=dev-02 \
--set "server.advertiseAddress={109.8.58.38}" \
--set server.modules.tunnel.security.enable=true \
--set server.modules.tunnel.security.httpServer="https://110.8.52.21:10002" \
--set agent.modules.tunnel.security.enable=true \
--set agent.modules.tunnel.security.httpServer="https://110.8.52.21:10002" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

### 手动配置

1. 配置 edgemesh-server 对应的 configmap

```yaml
apiVersion: v1
metadata:
  name: edgemesh-server-cfg
...
data:
  edgemesh-server.yaml: |
    ...
    modules:
      tunnel:
        # 插入如下内容
        security:
          enable: true
          # cloudcore 提供的 https 服务接口，用于证书签发
          httpServer: <cloudhub-https-addr>
```

完成上面的变更后，重新部署 edgemesh-server 就可以了。

2. 配置 edgemesh-agent 对应的 configmap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: edgemesh-agent-cfg
...
data:
  edgemesh-agent.yaml: |
    ...
    modules:
      tunnel:
        # 插入如下内容
        security:
          enable: true
          # cloudcore 提供的 https 服务接口，用于证书签发
          httpServer: <cloudhub-https-addr>
```

完成上面的变更后，重新部署 edgemesh-agent 就可以了。

:::tip
edgemesh-gateway 开启 security 特性的方式与 edgemesh-agent 相同
:::
