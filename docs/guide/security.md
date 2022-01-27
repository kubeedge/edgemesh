# EdgeMesh Security

The EdgeMesh service does not turn on security by default, and edgemesh-server establishes a connection with edgemesh-agent without any authentication. Here's the guide that how to turn on security (identity authentication, connection access, and communication encryption) for EdgeMesh services.

## Configuration

### Helm Configuration

Supports Helm deployment to directly enable the security feature. The httpServer needs to fill in the specific certificate authority. The specific command for deployment is as follows:

```shell
$ helm install edgemesh --set server.nodeName=dev-02 \
--set "server.advertiseAddress={109.8.58.38}" \
--set server.modules.tunnel.security.enable=true \
--set server.modules.tunnel.security.httpServer="https://ca-server-address" \
--set agent.modules.tunnel.security.enable=true \
--set agent.modules.tunnel.security.httpServer="https://ca-server-address" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

:::tip
CloudCore can act as a certificate authority, which requires KubeEdge version >= 1.8.2. Configuration example: httpServer="https://cloudcore-https-address:10002"
:::

### Manual configuration

1. config edgemesh-server's configmap

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
        # insert the following
        security:
          enable: true
          httpServer: <https://ca-server-address>
```

Once the changes above have been made, we can redeploy edgemesh-server directly.

2. config edgemesh-agent's configmap

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
        # insert the following
        security:
          enable: true
          httpServer: <https://ca-server-address>
```

Once the changes above have been made, we can redeploy edgemesh-agent directly.

:::tip
edgemesh-gateway enables the security feature in the same way as edgemesh-agent
:::
