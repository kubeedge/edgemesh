# EdgeMesh安全配置

## 目标
  EdgeMesh服务默认没有开启服务认证，edgemesh-server与edgemesh-agent建立连接时没有经过任何认证
  如下说明如何通过配置开启EdgeMesh服务之间的安全认证，通过如下的配置，EdgeMesh服务之间会开启身份认证、
  连接准入、通讯加密

### 依赖
KubeEdge版本 >= 1.8.2

### edgemesh-server配置
1. 配置edgemesh-server对应的configmap
```yaml
apiVersion: v1
metadata:
  name: edgemesh-server-cfg
......
data:
  edgemesh-server.yaml: |
    ......
    modules:
      tunnel:
        # 插入如下内容
        security:
          enable: true
          # cloudcore 提供的https服务接口，用于证书签发
          httpServer: <cloudhub-https-addr>
```
2. 重新部署
完成上面的变更后，重新部署edgemesh-server就可以了

### edgemesh-agent配置
1. 配置edgemesh-agent对应的configmap
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: edgemesh-agent-cfg
.......
data:
  edgemesh-agent.yaml: |
    ......
    modules:
      tunnel:
        # 插入如下内容
        security:
          enable: true
          # cloudcore 提供的https服务接口，用于证书签发
          httpServer: <cloudhub-https-addr>
  ```
2. 重新部署
完成上面的变更后，重新部署edgemesh-agent就可以了

### helm安装
master分支代码支持helm部署时直接开启security特性
假设cloudcore服务暴露的服务为110.8.52.21，则部署具体命令如下：
```yaml
helm install edgemesh --set server.nodeName=dev-02 \
--set "server.advertiseAddress={109.8.58.38}" \
--set server.modules.tunnel.security.enable=true \
--set server.modules.tunnel.security.httpServer="https://110.8.52.21:10002" \
--set server.image=kubeedge/edgemesh-server:latest \
--set agent.modules.tunnel.security.enable=true \
--set agent.modules.tunnel.security.httpServer="https://110.8.52.21:10002" \
--set agent.image=kubeedge/edgemesh-agent:latest \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```
