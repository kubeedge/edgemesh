# EdgeMesh Security

The EdgeMesh service does not turn on security by default, and edgemesh-server establishes a connection with edgemesh-agent without any authentication. Here's the guide that how to turn on security (identity authentication, connection access, and communication encryption) for EdgeMesh services.

## dependency

KubeEdge version >= 1.8.2

## Configuration

### Helm Configuration

Supports Helm deployment to directly enable the security feature. Assuming that the service exposed by the cloudcore service is 110.8.52.21, the specific deployment commands are as follows:

```shell
$ helm install edgemesh --set server.nodeName=dev-02 \
--set "server.advertiseAddress={109.8.58.38}" \
--set server.modules.tunnel.security.enable=true \
--set server.modules.tunnel.security.httpServer="https://110.8.52.21:10002" \
--set agent.modules.tunnel.security.enable=true \
--set agent.modules.tunnel.security.httpServer="https://110.8.52.21:10002" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

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
          # cloudcore's https api for crts sign apply
          httpServer: <cloudhub-https-addr>
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
          # cloudcore's https api for crt sign apply
          httpServer: <cloudhub-https-addr>
```

Once the changes above have been made, we can redeploy edgemesh-agent directly.

:::tip
edgemesh-gateway enables the security feature in the same way as edgemesh-agent
:::
