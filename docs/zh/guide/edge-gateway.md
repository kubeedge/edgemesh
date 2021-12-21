# 边缘网关

EdgeMesh 的边缘网关提供了通过网关的方式访问集群内部服务的能力，本章节会指导您从头部署一个边缘网关。

![edgemesh-ingress-gateway](/images/guide/em-ig.png)

## 部署

### Helm 部署

```shell
$ helm install edgemesh-gateway --set nodeName=<your node name> \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh-gateway.tgz
```

::: warning
请根据你的 K8s 集群设置 nodeName，否则 edgemesh-gateway 可能无法运行
:::

### 手动部署

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-gateway/
namespace/kubeedge unchanged
serviceaccount/edgemesh-gateway created
clusterrole.rbac.authorization.k8s.io/edgemesh-gateway created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-gateway created
configmap/edgemesh-gateway-cfg created
deployment.apps/edgemesh-gateway created
```

::: warning
请根据你的 K8s 集群设置 06-deployment.yaml 的 nodeName，否则 edgemesh-gateway 可能无法运行
:::

## HTTP 网关

**创建 Gateway 资源对象和路由规则 VirtualService**

```shell
$ kubectl apply -f examples/hostname-lb-random-gateway.yaml
deployment.apps/hostname-lb-edge created
service/hostname-lb-svc created
gateway.networking.istio.io/edgemesh-gateway created
destinationrule.networking.istio.io/hostname-lb-svc created
virtualservice.networking.istio.io/edgemesh-gateway-svc created
```

**查看 edgemesh-gateway 是否创建成功**

```shell
$ kubectl get gw
NAME               AGE
edgemesh-gateway   3m30s
```

**最后，使用 IP 和 Gateway 暴露的端口来进行访问**

```shell
$ curl 192.168.0.211:23333
```

## HTTPS 网关

**创建测试密钥文件**

```bash
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=kubeedge.io"
Generating a RSA private key
............+++++
.......................................................................................+++++
writing new private key to 'tls.key'
-----
```

**根据密钥文件创建 Secret 资源对象**

```bash
$ kubectl create secret tls gw-secret --key tls.key --cert tls.crt
secret/gw-secret created
```

**创建绑定了 Secret 的 Gateway 资源对象和路由规则 VirtualService**

```bash
$ kubectl apply -f examples/hostname-lb-random-gateway-tls.yaml
deployment.apps/hostname-lb-edge created
service/hostname-lb-svc created
gateway.networking.istio.io/edgemesh-gateway created
destinationrule.networking.istio.io/hostname-lb-svc created
virtualservice.networking.istio.io/edgemesh-gateway-svc created
```

**最后，使用证书进行 HTTPS 访问**

```bash
$ curl -k --cert ./tls.crt --key ./tls.key https://192.168.0.129:23333
```
