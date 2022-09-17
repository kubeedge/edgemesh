# EdgeMesh ingress gateway

Visit https://edgemesh.netlify.app/reference/config-items.html#helm-configuration for more configuration information.

访问 https://edgemesh.netlify.app/zh/reference/config-items.html#helm-配置 以了解更多的配置信息。

## Install

```
helm install edgemesh-gateway --namespace kubeedge \
--set psk=<your psk string> \
--set relayNodes[0].nodeName=<your node name>,relayNodes[0].advertiseAddress=<your advertise address list> \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh-gateway.tgz
```

**Install examples:**

You need to generate a PSK cipher first, please refer to: https://edgemesh.netlify.app/guide/security.html

Start with a relay node:
```
helm install edgemesh-gateway --namespace kubeedge \
--set psk=<your psk string> \
--set relayNodes[0].nodeName=k8s-master,relayNodes[0].advertiseAddress="{1.1.1.1}" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh-gateway.tgz
```

Start with two relay nodes:
```
helm install edgemesh-gateway --namespace kubeedge \
--set psk=<your psk string> \
--set relayNodes[0].nodeName=k8s-master,relayNodes[0].advertiseAddress="{1.1.1.1}" \
--set relayNodes[1].nodeName=ke-edge1,relayNodes[1].advertiseAddress="{2.2.2.2,3.3.3.3}" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh-gateway.tgz
```

## Uninstall

```
helm uninstall edgemesh-gateway -n kubeedge
```
