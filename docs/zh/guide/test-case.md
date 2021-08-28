# 测试用例

本章节里的所有测试用例都在目录 [examples](https://github.com/kubeedge/edgemesh/tree/main/examples) 下可找到对应文件。

## HTTP

在边缘节点上，部署支持 http 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/hostname.yaml
```

到边缘节点上，使用 curl 去访问相关服务，打印出容器的 hostname

```shell
$ curl hostname-lb-svc.edgemesh-test:12345
```

## TCP

在边缘节点1，部署支持 tcp 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/tcp-echo-service.yaml
```

在边缘节点2，使用 telnet 去访问相关服务

```shell
$ telnet tcp-echo-service.edgemesh-test 2701
```

## Websocket

在边缘节点1，部署支持 websocket 协议的容器应用和相关服务

```shell
$ kubectl apply -f examples/websocket-pod-svc.yaml
```

进入 websocket 的容器环境，并使用 client 去访问相关服务

```shell
$ docker exec -it 2a6ae1a490ae bash
$ ./client --addr ws-svc.edgemesh-test:12348
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

## 跨边云通信 :star:

处于 edgezone 的 busybox-edge 应用能够访问云上的 tcp-echo-cloud 应用，处于 cloudzone 的 busybox-cloud 能够访问边缘的 tcp-echo-edge 应用

```shell
$ kubectl apply -f examples/cloudzone.yaml
$ kubectl apply -f examples/edgezone.yaml
```

**云访问边**

```shell
$ kubectl -n cloudzone exec busybox-sleep-cloud -c busybox -i -t -- sh
/ # telnet tcp-echo-edge-svc.edgezone 2701
Welcome, you are connected to node ke-edge1.
Running on Pod tcp-echo-edge.
In namespace edgezone.
With IP address 172.17.0.2.
Service default.
I'm Cloud Buxybox
I'm Cloud Buxybox
```

**边访问云**

```shell
$ docker exec -it 4c57a4ff8974 sh
/ # telnet tcp-echo-cloud-svc.cloudzone 2701
Welcome, you are connected to node k8s-master.
Running on Pod tcp-echo-cloud.
In namespace cloudzone.
With IP address 10.244.0.8.
Service default.
I'm Edge Busybox
I'm Edge Busybox
```
