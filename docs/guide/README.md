# Getting Started

## Dependencies

[KubeEdge Dependencies](https://kubeedge.io/en/docs/#dependencies)

[KubeEdge >= v1.7.0](https://github.com/kubeedge/kubeedge/releases)

::: tip
- EdgeMesh isn't really depending on KubeEdge, it interacts with standard Kubernetes APIs only

- Regarding the fact that edge nodes may be isolated in different edge network, we are benefiting from "autonomic Kube-API endpoint" feature to simplify the setup
:::

## Prerequisites
- **Step 1**: Remove the taint of the K8s master nodes

```
kubectl taint nodes --all node-role.kubernetes.io/master-
```
If the application that needs to be proxied is not deployed on the K8s master nodes, the above steps can be omitted.

- **Step 2**: Modify KubeEdge Configuration

(1) Enable Local APIServer

On the cloud, open the dynamicController module, and restart cloudcore

```shell
$ vim /etc/kubeedge/config/cloudcore.yaml
modules:
  ..
  dynamicController:
    enable: true
..
```

```shell
# If cloudcore is not configured for systemd management, use the following command to restart (cloudcore < 1.10 is not configured for systemd management by default)
$ pkill cloudcore ; nohup /usr/local/bin/cloudcore > /var/log/kubeedge/cloudcore.log 2>&1 &

# If cloudcore is configured for systemd management, use the following command to restart
$ systemctl restart cloudcore

# If cloudcore uses containerized deployment, delete cloudcore's pods with `kubectl`
$ kubectl -n kubeedge delete pod <your cloudcore pod name>
```

At the edge node, open metaServer module (if your KubeEdge < 1.8.0, you also need to close edgeMesh module)

```shell
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ..
  edgeMesh:
    enable: false
  ..
  metaManager:
    metaServer:
      enable: true
..
```

(2) Configure clusterDNS and clusterDomain

At the edge node, configure clusterDNS, clusterDomain and restart edgecore

```shell
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ..
  edged:
    clusterDNS: 169.254.96.16
    clusterDomain: cluster.local
..
```

```shell
$ systemctl restart edgecore
```

::: tip
The value '169.254.96.16' set by clusterDNS comes from the default value of bridgeDeviceIP in [commonConfig](https://edgemesh.netlify.app/reference/config-items.html#edgemesh-agent-cfg), if you need to modify it, please keep the two consistent.
:::

(3) Check it out

At the edge node, check if Local APIServer works

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

::: warning
If the return value is an empty list, or the response takes a long time (close to 10s) to get the return value, your configuration may be wrong, please check again.
:::

## Helm Installation

- **Step 1**: Install Charts

Make sure you have installed Helm 3, then refer to: [Helm Deployment EdgeMesh Guide](https://github.com/kubeedge/edgemesh/blob/main/build/helm/edgemesh/README.md)

- **Step 2**: Check it out

```shell
$ helm ls -A
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART           APP VERSION
edgemesh        kubeedge        1               2022-09-18 12:21:47.097801805 +0800 CST deployed        edgemesh-0.1.0  latest

$ kubectl get all -n kubeedge -o wide
NAME                       READY   STATUS    RESTARTS   AGE   IP              NODE         NOMINATED NODE   READINESS GATES
pod/edgemesh-agent-7gf7g   1/1     Running   0          39s   192.168.0.71    k8s-node1    <none>           <none>
pod/edgemesh-agent-fwf86   1/1     Running   0          39s   192.168.0.229   k8s-master   <none>           <none>
pod/edgemesh-agent-twm6m   1/1     Running   0          39s   192.168.5.121   ke-edge2     <none>           <none>
pod/edgemesh-agent-xwxlp   1/1     Running   0          39s   192.168.5.187   ke-edge1     <none>           <none>

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE   CONTAINERS       IMAGES                           SELECTOR
daemonset.apps/edgemesh-agent   4         4         4       4            4           <none>          39s   edgemesh-agent   kubeedge/edgemesh-agent:latest   k8s-app=kubeedge,kubeedge=edgemesh-agent
```

## Manual Installation

- **Step 1**: Download EdgeMesh

```shell
$ git clone https://github.com/kubeedge/edgemesh.git
$ cd edgemesh
```

- **Step 2**: Create CRDs

```shell
$ kubectl apply -f build/crds/istio/
customresourcedefinition.apiextensions.k8s.io/destinationrules.networking.istio.io created
customresourcedefinition.apiextensions.k8s.io/gateways.networking.istio.io created
customresourcedefinition.apiextensions.k8s.io/virtualservices.networking.istio.io created
```

- **Step 3**: Deploy edgemesh-agent

```shell
$ kubectl apply -f build/agent/resources/
serviceaccount/edgemesh-agent created
clusterrole.rbac.authorization.k8s.io/edgemesh-agent created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-agent created
configmap/edgemesh-agent-cfg created
configmap/edgemesh-agent-psk created
daemonset.apps/edgemesh-agent created
```

::: tip
Please set the relayNodes in build/agent/resources/04-configmap.yaml according to your K8s cluster and regenerate the PSK cipher.
:::

- **Step 4**: Check it out

```shell
$ kubectl get all -n kubeedge -o wide
NAME                       READY   STATUS    RESTARTS   AGE   IP              NODE         NOMINATED NODE   READINESS GATES
pod/edgemesh-agent-7gf7g   1/1     Running   0          39s   192.168.0.71    k8s-node1    <none>           <none>
pod/edgemesh-agent-fwf86   1/1     Running   0          39s   192.168.0.229   k8s-master   <none>           <none>
pod/edgemesh-agent-twm6m   1/1     Running   0          39s   192.168.5.121   ke-edge2     <none>           <none>
pod/edgemesh-agent-xwxlp   1/1     Running   0          39s   192.168.5.187   ke-edge1     <none>           <none>

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE   CONTAINERS       IMAGES                           SELECTOR
daemonset.apps/edgemesh-agent   4         4         4       4            4           <none>          39s   edgemesh-agent   kubeedge/edgemesh-agent:latest   k8s-app=kubeedge,kubeedge=edgemesh-agent
```
