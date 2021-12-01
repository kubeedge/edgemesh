# EdgeMesh app

Visit https://github.com/kubeedge/edgemesh for more information.

## Install

```
helm install edgemesh \
    --set server.nodeName=<your node name> --set "server.advertiseAddress=<your advertise address list>" .
```

**Install examples:**
```
helm install edgemesh \
    --set server.nodeName=k8s-node1 --set "server.advertiseAddress={119.8.211.54}" .
```

## Uninstall

```
helm uninstall edgemesh
```
