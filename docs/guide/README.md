# Getting Started

## Dependencies

[KubeEdge Dependencies](https://kubeedge.io/en/docs/#dependencies)

[KubeEdge >= v1.7.0](https://github.com/kubeedge/kubeedge/releases)

::: tip
- EdgeMesh isn't really depending on KubeEdge, it interacts with standard Kubernetes APIs only

- Regarding the fact that edge nodes may be isolated in different edge network, we are benefiting from [Edge Kube-API Endpoint](../guide/edge-kube-api.md) feature to simplify the setup
:::

## Prerequisites

- **Step 1**: Remove the taint of the K8s master nodes

```shell
$ kubectl taint nodes --all node-role.kubernetes.io/master-
```
If the application that needs to be proxied is not deployed on the K8s master nodes, the above steps can be omitted.

- **Step 2**: Add filter labels to Kubernetes API services

```shell
$ kubectl label services kubernetes service.edgemesh.kubeedge.io/service-proxy-name=""
```

Normally you don't want EdgeMesh to proxy the Kubernetes API service, so you need to add a filter label to it. For more information, please refer to [Service Filter](../advanced/hybird-proxy.md#service-filter).

- **Step 3**: Enable KubeEdge's Edge Kube-API Endpoint Service

Please refer to the documentation [Edge Kube-API Endpoint](../guide/edge-kube-api.md#quick-start) to enable this service.

## Install

We provide two ways to install EdgeMesh, you can choose one to deploy EdgeMesh according to your own situation.

### Helm Install

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

### Manual Install

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
