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
        enableSecurity: true
        ACL:
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
        enableSecurity: true
        ACL:
          # cloudcore 提供的https服务接口，用于证书签发
          httpServer: <cloudhub-https-addr>
  ```
2. 重新部署
完成上面的变更后，重新部署edgemesh-agent就可以了