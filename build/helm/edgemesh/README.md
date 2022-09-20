# EdgeMesh app

Visit https://edgemesh.netlify.app/reference/config-items.html#helm-configuration for more configuration information.

访问 https://edgemesh.netlify.app/zh/reference/config-items.html#helm-配置 以了解更多的配置信息。

## Install

```
helm install edgemesh --namespace kubeedge \
--set agent.psk=<your psk string> \
--set agent.relayNodes[0].nodeName=<your node name>,agent.relayNodes[0].advertiseAddress=<your advertise address list> \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

**Install examples:**

You need to generate a PSK cipher first, please refer to: https://edgemesh.netlify.app/guide/security.html

Start with a relay node:
```
helm install edgemesh --namespace kubeedge \
--set agent.psk=<your psk string> \
--set agent.relayNodes[0].nodeName=k8s-master,agent.relayNodes[0].advertiseAddress="{1.1.1.1}" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

Start with two relay nodes:
```
helm install edgemesh --namespace kubeedge \
--set agent.psk=<your psk string> \
--set agent.relayNodes[0].nodeName=k8s-master,agent.relayNodes[0].advertiseAddress="{1.1.1.1}" \
--set agent.relayNodes[1].nodeName=ke-edge1,agent.relayNodes[1].advertiseAddress="{2.2.2.2,3.3.3.3}" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

## Uninstall

```
helm uninstall edgemesh -n kubeedge
```
