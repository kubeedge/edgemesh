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
        enableSecurity: true
        ACL:
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
        enableSecurity: true
        ACL:
          # cloudcore's https api for crt sign apply
          httpServer: <cloudhub-https-addr>
```
2. redeploy
Once the changes above have been made, we can redeploy edgemesh-agent directly