# Edge Gateway

Edge Gateway provides the ability to access the internal services of the cluster through the gateway. This chapter will guide you to deploy an edge gateway from scratch.

![edgemesh-ingress-gateway](/images/guide/em-ig.png)

## Deploy

```shell
$ kubectl apply -f build/agent/kubernetes/edgemesh-gateway/02-configmap.yaml
configmap/edgemesh-gateway-cfg created
$ kubectl apply -f build/agent/kubernetes/edgemesh-gateway/03-deployment.yaml
deployment.apps/edgemesh-gateway created
```

::: tip
Edge Gateway and EdgeMesh-Agent use the same [docker image](https://hub.docker.com/r/kubeedge/edgemesh-agent), with only minor differences in configuration.
:::

## HTTP Gateway

**Create 'Gateway' and 'VirtualService'**

```shell
$ kubectl apply -f examples/hostname-lb-random-gateway.yaml
pod/hostname-lb-edge2 created
pod/hostname-lb-edge3 created
service/hostname-lb-svc created
gateway.networking.istio.io/edgemesh-gateway configured
destinationrule.networking.istio.io/hostname-lb-edge created
virtualservice.networking.istio.io/edgemesh-gateway-svc created
```

**Check if the edgemesh-gateway is successfully created**

```shell
$ kubectl get gw -n edgemesh-test
NAME               AGE
edgemesh-gateway   3m30s
```

**Finally, use the IP and the port exposed by the Gateway to access**

```shell
$ curl 192.168.0.211:12345
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
$ kubectl create secret tls gw-secret --key tls.key --cert tls.crt -n edgemesh-test
secret/gw-secret created
```

**Create a Secret-bound 'Gateway' and routing rules 'VirtualService'**

```bash
$ kubectl apply -f examples/hostname-lb-random-gateway-tls.yaml
pod/hostname-lb-edge2 created
pod/hostname-lb-edge3 created
service/hostname-lb-svc created
gateway.networking.istio.io/edgemesh-gateway configured
destinationrule.networking.istio.io/hostname-lb-edge created
virtualservice.networking.istio.io/edgemesh-gateway-svc created
```

**Finally, use the certificate for a HTTPS access**

```bash
$ curl -k --cert ./tls.crt --key ./tls.key https://192.168.0.129:12345
```
