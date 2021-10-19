# EdgeMesh ingress gateway

Visit https://github.com/kubeedge/edgemesh for more information.

## Install

```
helm install edgemesh-gateway \
    --set nodeName=<your node name> .
```

**Install examples:**
```
helm install edgemesh-gateway \
    --set nodeName=ke-edge1 .
```

## Uninstall

```
helm uninstall edgemesh-gateway
```
