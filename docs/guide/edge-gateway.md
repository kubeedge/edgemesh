# Edge Gateway

Edge Gateway provides the ability to access the internal services of the cluster through the gateway. This chapter will guide you to deploy an edge gateway from scratch.

![edgemesh-ingress-gateway](/images/guide/em-ig.png)

## Deploy

Before deploying the edgemesh-gateway, make sure that edgemesh-server and edgemesh-agent have been deployed successfully.

### Helm Deploy

```shell
$ helm install edgemesh-gateway --set nodeName=<your node name> \
https://raw.githubusercontent.com/kubeedge/edgemesh/main/build/helm/edgemesh-gateway.tgz
```

::: warning
Please set the value of  nodeName according to your K8s cluster, otherwise edgemesh-gateway may not run
:::

### Manual Deploy

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-gateway/
serviceaccount/edgemesh-gateway created
clusterrole.rbac.authorization.k8s.io/edgemesh-gateway created
clusterrolebinding.rbac.authorization.k8s.io/edgemesh-gateway created
configmap/edgemesh-gateway-cfg created
deployment.apps/edgemesh-gateway created
```

::: warning
Please set the value of 05-deployment.yaml's nodeName according to your K8s cluster, otherwise edgemesh-gateway may not run
:::

## HTTP Gateway

**Create 'Gateway' and 'VirtualService'**

```shell
$ kubectl apply -f examples/hostname-lb-random-gateway.yaml
deployment.apps/hostname-lb-edge created
service/hostname-lb-svc created
gateway.networking.istio.io/edgemesh-gateway created
destinationrule.networking.istio.io/hostname-lb-svc created
virtualservice.networking.istio.io/edgemesh-gateway-svc created
```

**Check if the edgemesh-gateway is successfully created**

```shell
$ kubectl get gw
NAME               AGE
edgemesh-gateway   3m30s
```

**Finally, use the IP and the port exposed by the Gateway to access**

```shell
$ curl 192.168.0.211:23333
```

## HTTPS Gateway

**Create a test key file**

```bash
$ openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=kubeedge.io"
Generating a RSA private key
............+++++
.......................................................................................+++++
writing new private key to 'tls.key'
-----
```

**Create a 'Secret' according to the key file**

```bash
$ kubectl create secret tls gw-secret --key tls.key --cert tls.crt
secret/gw-secret created
```

**Create a Secret-bound 'Gateway' and routing rules 'VirtualService'**

```bash
$ kubectl apply -f examples/hostname-lb-random-gateway-tls.yaml
deployment.apps/hostname-lb-edge created
service/hostname-lb-svc created
gateway.networking.istio.io/edgemesh-gateway created
destinationrule.networking.istio.io/hostname-lb-svc created
virtualservice.networking.istio.io/edgemesh-gateway-svc created
```

**Finally, use the certificate for a HTTPS access**

```bash
$ curl -k --cert ./tls.crt --key ./tls.key https://192.168.0.211:23333
```
