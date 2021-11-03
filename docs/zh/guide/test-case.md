# 测试用例

本章节里的所有测试用例都在目录 [examples](https://github.com/kubeedge/edgemesh/tree/main/examples) 下可找到对应文件。

## HTTP

部署支持 http 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/hostname.yaml
```

在边缘节点，使用 `curl` 去访问相关服务，打印出容器的 hostname

```shell
$ curl hostname-svc.default:12345
```

## TCP

部署支持 tcp 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/tcp-echo-service.yaml
```

在边缘节点，使用 `telnet` 去访问相关服务

```shell
$ telnet tcp-echo-service.default 2701
```

## Websocket

部署支持 websocket 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/websocket.yaml
```

在边缘节点，进入 websocket 的容器环境，并使用 client 去访问相关服务

```shell
$ WEBSOCKET_CID=$(docker ps | grep k8s_ws_ws-edge | awk '{print $1}')
$ docker exec -it $WEBSOCKET_CID bash
$ ./client --addr ws-svc.default:12348
```

## 负载均衡

使用 DestinationRule 中的 loadBalancer 属性来选择不同的负载均衡模式

```shell
$ vim examples/hostname-lb-random.yaml
spec
..
  trafficPolicy:
    loadBalancer:
      simple: RANDOM
..
```

在边缘节点，使用 `curl` 去访问相关服务，你将看到多个 hostname-edge 被随机访问

```shell
$ curl hostname-lb-svc.default:12345
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
