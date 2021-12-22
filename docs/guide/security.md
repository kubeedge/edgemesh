# EdgeMesh Security

## Goal
The EdgeMesh service does not turn on security by default, and edgemesh-server
establishes a connection with edgemesh-agent without any authentication.
Here's the guide that how to turn on security (identity authentication,
connection access, and communication encryption) for EdgeMesh services.

### dependency
KubeEdge version >= 1.8.2

### edgemesh-server configuration
1. config edgemesh-server's configmap
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
        # insert the following
        security:
          enable: true
          # cloudcore's https api for crts sign apply
          httpServer: <cloudhub-https-addr>
```
2. redeploy
Once the changes above have been made, we can redeploy edgemesh-server directly

### edgemesh-agent configuration
1. config edgemesh-agent's configmap
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
        # insert the following
        security:
          enable: true
          # cloudcore's https api for crt sign apply
          httpServer: <cloudhub-https-addr>
```
2. redeploy
Once the changes above have been made, we can redeploy edgemesh-agent directly


### helm installation
The master branch code supports directly enabling the security feature using helm.
Assume that the ip of cloudcore service is 110.8.52.21, the installation commands are as follows:
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