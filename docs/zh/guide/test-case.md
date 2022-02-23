# 测试用例

本章节里的所有测试用例都在目录 [examples](https://github.com/kubeedge/edgemesh/tree/main/examples) 下可找到对应文件。

## 准备工作

- **步骤1**: 部署 EdgeMesh

请参考 [快速上手](./getting-started.md) 完成 EdgeMesh 的部署

- **步骤2**: 部署测试容器

```shell
$ kubectl apply -f examples/test-pod.yaml
pod/alpine-test created
pod/websocket-test created
```

## HTTP

部署支持 http 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/hostname.yaml
deployment.apps/hostname-edge created
service/hostname-svc created
```

进入测试容器，并使用 `curl` 去访问相关服务

```shell
$ kubectl exec -it alpine-test -- sh
(在容器环境内)
/ # curl hostname-svc:12345
hostname-edge-5c75d56dc4-rq57t
```

## HTTPS

部署支持 https 协议的容器应用和相关服务

```shell
$ ./examples/nginx-https/tools.sh install
...
Getting Private key
Getting CA Private Key
secret/nginxsecret created
configmap/nginxconfigmap created
deployment.apps/nginx-https created
service/nginx-https created
create https example success!
```

进入测试容器，并使用 `curl` 去访问相关服务

```shell
$ kubectl exec -it alpine-test -- sh
(在容器环境内)
/ # curl -k --cert client.crt --key client.key https://nginx-https
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
...

(也可以使用外部域名去访问相关服务)
/ # curl --cacert rootCA.crt --cert client.crt --key client.key https://my-nginx.com
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
...
```

::: details
examples/nginx-https/tools.sh 脚本功能：
1. 生成自签根证书，服务器、客户端的证书和私钥
2. 创建 nginx-https 相关的 secret，configmap，deployment 和 service
3. 往 alpine-test 内复制证书和私钥
4. 往 alpine-test 内的 /etc/hosts 文件写入 IP 与域名 (my-nginx.com) 的映射

备注：使用 tools.sh 脚本的 cleanup 命令可清空上述创建的所有资源，以及还原对 alpine-test 的修改
:::

## TCP

部署支持 tcp 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/tcp-echo-service.yaml
deployment.apps/tcp-echo-deployment created
service/tcp-echo-service created
```

进入测试容器，并使用 `telnet` 去访问相关服务

```shell
$ kubectl exec -it alpine-test -- sh
(在容器环境内)
/ # telnet tcp-echo-service 2701
Welcome, you are connected to node ke-edge1.
Running on Pod tcp-echo-deployment-66457b769-7zgqb.
In namespace default.
With IP address 172.17.0.2.
Service default.
```

## Websocket

部署支持 websocket 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/websocket.yaml
deployment.apps/ws-edge created
service/ws-svc created
```

进入测试容器，并使用 websocket `client` 去访问相关服务

```shell
$ kubectl exec -it websocket-test -- sh
(在容器环境内)
/ # ./client --addr ws-svc:12348
connecting to ws://ws-svc.default:12348/echo
recv: 2021-12-02 03:42:20.191695384 +0000 UTC m=+1.004526202
recv: 2021-12-02 03:42:21.191724176 +0000 UTC m=+2.004554995
recv: 2021-12-02 03:42:22.191725321 +0000 UTC m=+3.004556159
```

## UDP

部署支持 udp 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/hostname-udp.yaml
deployment.apps/hostname-edge created
service/hostname-udp-svc created
```

进入测试容器，并使用 `nc` 去访问相关服务

```shell
$ kubectl exec -it alpine-test -- sh
(在容器环境内)
/ # nc -u hostname-udp-svc 12345
hostname-edge-5cd47b65d5-8zg27
```

## 负载均衡

部署配置了 `random` 负载均衡策略的容器应用和相关服务

```shell
$ kubectl apply -f examples/hostname-lb-random.yaml
deployment.apps/hostname-lb-edge created
service/hostname-lb-svc created
destinationrule.networking.istio.io/hostname-lb-svc created
```

:::tip
EdgeMesh 使用了 DestinationRule 中的 loadBalancer 属性来选择不同的负载均衡策略。使用 DestinationRule 时，要求 DestinationRule 的名字与相应的 Service 的名字要一致，EdgeMesh 会根据 Service 的名字来确定同命名空间下面的 DestinationRule
:::

进入测试容器，并多次使用 `curl` 去访问相关服务，你将看到多个 hostname-edge 被随机的访问

```shell
$ kubectl exec -it alpine-test -- sh
(在容器环境内)
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-w82nw
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-xjp86
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-xjp86
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-iq39z
```

## 跨边云通信 :star:

处于 edgezone 的 busybox-edge 应用能够访问云上的 tcp-echo-cloud 应用，处于 cloudzone 的 busybox-cloud 应用能够访问边缘的 tcp-echo-edge 应用

**部署**

```shell
$ kubectl apply -f examples/cloudzone.yaml
namespace/cloudzone created
deployment.apps/tcp-echo-cloud created
service/tcp-echo-cloud-svc created
deployment.apps/busybox-sleep-cloud created
```

```
$ kubectl apply -f examples/edgezone.yaml
namespace/edgezone created
deployment.apps/tcp-echo-edge created
service/tcp-echo-edge-svc created
deployment.apps/busybox-sleep-edge created
```

**云访问边**

```shell
$ BUSYBOX_POD=$(kubectl get all -n cloudzone | grep pod/busybox | awk '{print $1}')
$ kubectl -n cloudzone exec $BUSYBOX_POD -c busybox -i -t -- sh
$ telnet tcp-echo-edge-svc.edgezone 2701
Welcome, you are connected to node ke-edge1.
Running on Pod tcp-echo-edge.
In namespace edgezone.
With IP address 172.17.0.2.
Service default.
Hello Edge, I am Cloud.
Hello Edge, I am Cloud.
```

**边访问云**

在边缘节点，使用 `telnet` 去访问相关服务

```shell
$ BUSYBOX_CID=$(docker ps | grep k8s_busybox_busybox-sleep-edge | awk '{print $1}')
$ docker exec -it $BUSYBOX_CID sh
$ telnet tcp-echo-cloud-svc.cloudzone 2701
Welcome, you are connected to node k8s-master.
Running on Pod tcp-echo-cloud.
In namespace cloudzone.
With IP address 10.244.0.8.
Service default.
Hello Cloud, I am Edge.
Hello Cloud, I am Edge.
```
