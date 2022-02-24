# Getting Started

## Dependencies

[KubeEdge Dependencies](https://kubeedge.io/en/docs/#dependencies)

[KubeEdge v1.7+](https://github.com/kubeedge/kubeedge/releases)

::: tip
- EdgeMesh isn't really depending on KubeEdge, it interacts with standard Kubernetes APIs only

- Regarding the fact that edge nodes may be isolated in different edge network, we are benefiting from "autonomic Kube-API endpoint" feature to simplify the setup
:::

## Helm Installation

- **Step 1**: Modify KubeEdge Configuration

Refer to [Manual Installation-Step 3](#step3) to modify the configuration of KubeEdge.

- **Step 2**: Install Charts

Make sure you have installed Helm 3

```
$ helm install edgemesh \
--set server.nodeName=<your node name> \
--set server.advertiseAddress="{your edgemesh server adveritise address list, such as node eip}" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

server.nodeName specifies the node deployed by edgemesh-server, and server.advertiseAddress specifies the edgemesh-server
advertise address list and use commas to separate IP, such as `{119.8.211.54,100.10.1.4}`.
The server.advertiseAddress can be omitted, because edgemesh-server will automatically detect and configure the advertiseAddress list, but it is not guaranteed to be correct.

**Example：**

```shell
$ helm install edgemesh \
--set server.nodeName=k8s-node1 \
--set server.advertiseAddress="{119.8.211.54}" \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh.tgz
```

::: warning
Please set server.nodeName and server.advertiseAddress according to your K8s cluster, otherwise edgemesh-server may not run
:::

- **Step 3**: Check it out

```shell
$ helm ls
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART           APP VERSION
edgemesh        default         1               2021-11-01 23:30:02.927346553 +0800 CST deployed        edgemesh-0.1.0  latest
```

```shell
$ kubectl get all -n kubeedge -o wide
NAME                                   READY   STATUS    RESTARTS   AGE   IP             NODE         NOMINATED NODE   READINESS GATES
pod/edgemesh-agent-2m9pq               1/1     Running   0          16m   192.168.22.3   k8s-node02   <none>           <none>
pod/edgemesh-agent-479rz               1/1     Running   0          16m   192.168.22.2   k8s-node01   <none>           <none>
pod/edgemesh-agent-8cd2j               1/1     Running   0          16m   192.168.22.5   ke-edge2     <none>           <none>
pod/edgemesh-agent-phfln               1/1     Running   0          16m   192.168.22.4   ke-edge1     <none>           <none>
pod/edgemesh-server-74dc5c67dc-kdf2b   1/1     Running   0          22m   192.168.22.2   k8s-node01   <none>           <none>

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE   CONTAINERS       IMAGES                           SELECTOR
daemonset.apps/edgemesh-agent   4         4         4       4            4           <none>          16m   edgemesh-agent   kubeedge/edgemesh-agent:latest   k8s-app=kubeedge,kubeedge=edgemesh-agent


NAME                              READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS        IMAGES                            SELECTOR
deployment.apps/edgemesh-server   1/1     1            1           22m   edgemesh-server   kubeedge/edgemesh-server:latest   k8s-app=kubeedge,kubeedge=edgemesh-server


NAME                                         DESIRED   CURRENT   READY   AGE   CONTAINERS        IMAGES                            SELECTOR
replicaset.apps/edgemesh-server-74dc5c67dc   1         1         1       22m   edgemesh-server   kubeedge/edgemesh-server:latest   k8s-app=kubeedge,kubeedge=edgemesh-server,pod-template-hash=74dc5c67dc
```

## Manual Installation

- **Step 1**: Download EdgeMesh

```shell
$ git clone https://github.com/kubeedge/edgemesh.git
$ cd edgemesh
```

<a name="step3"></a>
- **Step 2**: Create CRDs

```shell
$ kubectl apply -f build/crds/istio/
customresourcedefinition.apiextensions.k8s.io/destinationrules.networking.istio.io created
customresourcedefinition.apiextensions.k8s.io/gateways.networking.istio.io created
customresourcedefinition.apiextensions.k8s.io/virtualservices.networking.istio.io created
```

- **Step 3**: Modify KubeEdge Configuration

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
# If cloudcore is not configured for systemd management, use the following command to restart (cloudcore is not configured for systemd management by default)
$ pkill cloudcore ; nohup /usr/local/bin/cloudcore > /var/log/kubeedge/cloudcore.log 2>&1 &

# If cloudcore is configured for systemd management, use the following command to restart
$ systemctl restart cloudcore
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
The value '169.254.96.16' set by clusterDNS comes from the default value of dummyDeviceIP in [commonConfig](../reference/config-items.md#table-1-2-commonconfig), if you need to modify it, please keep the two consistent
:::

(3) Check it out

At the edge node, check if Local APIServer works

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

- **Step 4**: Deploy edgemesh-server

```shell
$ kubectl apply -f build/server/edgemesh/
serviceaccount/edgemesh-server created
clusterrole.rbac.authorization.k8s.io/edgemesh-server created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-server created
configmap/edgemesh-server-cfg created
deployment.apps/edgemesh-server created
```

::: warning
Please set the value of build/server/edgemesh/04-configmap.yaml's advertiseAddress and build/server/edgemesh/05-deployment.yaml's nodeName according to your K8s cluster, otherwise edgemesh-server may not run
:::

- **Step 5**: Deploy edgemesh-agent

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-agent/
serviceaccount/edgemesh-agent created
clusterrole.rbac.authorization.k8s.io/edgemesh-agent created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-agent created
configmap/edgemesh-agent-cfg created
daemonset.apps/edgemesh-agent created
```

- **Step 6**: Check it out

```shell
$ kubectl get all -n kubeedge -o wide
NAME                                   READY   STATUS    RESTARTS   AGE   IP             NODE         NOMINATED NODE   READINESS GATES
pod/edgemesh-agent-2m9pq               1/1     Running   0          16m   192.168.22.3   k8s-node02   <none>           <none>
pod/edgemesh-agent-479rz               1/1     Running   0          16m   192.168.22.2   k8s-node01   <none>           <none>
pod/edgemesh-agent-8cd2j               1/1     Running   0          16m   192.168.22.5   ke-edge2     <none>           <none>
pod/edgemesh-agent-phfln               1/1     Running   0          16m   192.168.22.4   ke-edge1     <none>           <none>
pod/edgemesh-server-74dc5c67dc-kdf2b   1/1     Running   0          22m   192.168.22.2   k8s-node01   <none>           <none>

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE   CONTAINERS       IMAGES                           SELECTOR
daemonset.apps/edgemesh-agent   4         4         4       4            4           <none>          16m   edgemesh-agent   kubeedge/edgemesh-agent:latest   k8s-app=kubeedge,kubeedge=edgemesh-agent


NAME                              READY   UP-TO-DATE   AVAILABLE   AGE   CONTAINERS        IMAGES                            SELECTOR
deployment.apps/edgemesh-server   1/1     1            1           22m   edgemesh-server   kubeedge/edgemesh-server:latest   k8s-app=kubeedge,kubeedge=edgemesh-server


NAME                                         DESIRED   CURRENT   READY   AGE   CONTAINERS        IMAGES                            SELECTOR
replicaset.apps/edgemesh-server-74dc5c67dc   1         1         1       22m   edgemesh-server   kubeedge/edgemesh-server:latest   k8s-app=kubeedge,kubeedge=edgemesh-server,pod-template-hash=74dc5c67dc
```
