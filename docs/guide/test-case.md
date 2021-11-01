# Test Case

All the test cases in this chapter can find the corresponding files in the directory [examples](https://github.com/kubeedge/edgemesh/tree/main/examples).

## HTTP

At the edge node, deploy a HTTP container application, and relevant service

```shell
$ kubectl apply -f examples/hostname.yaml
```

Go to that edge node, use ‘curl’ to access the service, and print out the hostname of the container

```shell
$ curl hostname-lb-svc.edgemesh-test:12345
```

## TCP

At the edge node 1, deploy a TCP container application, and relevant service

```shell
$ kubectl apply -f examples/tcp-echo-service.yaml
```

At the edge node 2, use ‘telnet’ to access the service

```shell
$ telnet tcp-echo-service.edgemesh-test 2701
```

## Websocket

At the edge node 1, deploy a websocket container application, and relevant service

```shell
$ kubectl apply -f examples/websocket-pod-svc.yaml
```

Enter the container, and use ./client to access the service

```shell
$ docker exec -it 2a6ae1a490ae bash
$ ./client --addr ws-svc.edgemesh-test:12348
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

## Cross-Edge-Cloud :star:

The busybox-edge in the edgezone can access the tcp-echo-cloud on the cloud, and the busybox-cloud in the cloudzone can access the tcp-echo-edge on the edge

```shell
$ kubectl apply -f examples/cloudzone.yaml
$ kubectl apply -f examples/edgezone.yaml
```

**Cloud access edge**

```shell
$ kubectl -n cloudzone exec busybox-sleep-cloud -c busybox -i -t -- sh
/ # telnet tcp-echo-edge-svc.edgezone 2701
Welcome, you are connected to node ke-edge1.
Running on Pod tcp-echo-edge.
In namespace edgezone.
With IP address 172.17.0.2.
Service default.
I'm Cloud Busybox
I'm Cloud Busybox
```

**Edge access cloud**

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
