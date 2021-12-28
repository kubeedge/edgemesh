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
> If `--set nodeName=ke-edge1 .` is not set, nodeAffinity or nodeSelector will be used for scheduling. Please add the value of at least one of these two fields

## Uninstall

```
helm uninstall edgemesh-gateway
```
