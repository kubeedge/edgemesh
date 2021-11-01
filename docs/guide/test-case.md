# Test Case

All the test cases in this chapter can find the corresponding files in the directory [examples](https://github.com/kubeedge/edgemesh/tree/main/examples).

## HTTP

Deploy a HTTP container application, and relevant service

```shell
$ kubectl apply -f examples/hostname.yaml
```

At the edge node, use `curl` to access the service, and print out the hostname of the container

```shell
$ curl hostname-svc.default:12345
```

## TCP

Deploy a TCP container application, and relevant service

```shell
$ kubectl apply -f examples/tcp-echo-service.yaml
```

At the edge node, use `telnet` to access the service

```shell
$ telnet tcp-echo-service.default 2701
```

## Websocket

Deploy a websocket container application, and relevant service

```shell
$ kubectl apply -f examples/websocket.yaml
```

At the edge node, enter the container, and use ./client to access the service

```shell
$ WEBSOCKET_CID=$(docker ps | grep k8s_ws_ws-edge | awk '{print $1}')
$ docker exec -it $WEBSOCKET_CID bash
$ ./client --addr ws-svc.default:12348
```

## Load Balance

Use the 'loadBalancer' in 'DestinationRule' to select LB modes

```shell
$ vim examples/hostname-lb-random.yaml
spec
..
  trafficPolicy:
    loadBalancer:
      simple: RANDOM
..
```

At the edge node, use `curl` to access the service, you will see that multiple hostname-edge are randomly accessed

```shell
$ curl hostname-lb-svc.default:12345
```

## Cross-Edge-Cloud :star:

The busybox-edge in the edgezone can access the tcp-echo-cloud on the cloud, and the busybox-cloud in the cloudzone can access the tcp-echo-edge on the edge

**Deploy**

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

**Cloud access edge**

```shell
$ kubectl -n cloudzone exec busybox-sleep-cloud -c busybox -i -t -- sh
$ telnet tcp-echo-edge-svc.edgezone 2701
Welcome, you are connected to node ke-edge1.
Running on Pod tcp-echo-edge.
In namespace edgezone.
With IP address 172.17.0.2.
Service default.
Hello Edge, I am Cloud.
Hello Edge, I am Cloud.
```

**Edge access cloud**

At the edge node, use `telnet` to access the service

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
