# Hybrid-NAT

## Goal

When both `kube-proxy` and `edgemesh` are deployed in the cluster, the compatibility of the two is ensured by the following methods:

When `edgemesh-agent` is started after `kube-proxy`, the customized `EDGE-MESH` chain will be inserted into the `PREROUTING` and `OUTPUT` chains in the nat table of the host iptables. This enables `edgemesh-agent` to hijack service traffic in priority over `kube-proxy` to achieve cloud-edge communication traffic forwarding.

```shell
Chain PREROUTING (policy ACCEPT 2167 packets, 135K bytes)
num   pkts bytes target     prot opt in     out     source               destination
1    2655K  338M EDGE-MESH  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* edgemesh root chain */
2    2656K  338M KUBE-SERVICES  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* kubernetes service portals */

Chain OUTPUT (policy ACCEPT 3825 packets, 250K bytes)
num   pkts bytes target     prot opt in     out     source               destination
1    7919K  508M EDGE-MESH  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* edgemesh root chain */
2    4359K  294M KUBE-SERVICES  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* kubernetes service portals */
```


## Traffic Proxy

The `edgemesh-agent` intercepts and forwards all services accessed through the Cluster IP by default. However, the traffic forwarding process of `edgemesh-agent` will cause some performance loss in the user mode. In order to improve the efficiency of traffic forwarding, we support the ignore setting of the traffic proxy of the service, so that it uses the proxy of `kube-proxy` for traffic forwarding.

**Services ignored by default**

There are two cluster services that are ignored by default, namely the kube-dns(coredns) service in the kube-system namespace and the kubernetes service in the default namespace.

Take the kube-dns service as an example: the program will set the ignore according to the kube-dns service with `label: k8s-app=kube-dns`, so that it can bypass `PREROUTING -> EDGE-MESH`. Then `RETURN` back to the `KUBE-SERVICES` chain set by kube-proxy, and continue traffic forwarding by the kernel's netfilter.

```shell
$ kubectl get svc -owide -nkube-system --show-labels|grep coredns
coredns   ClusterIP   10.10.0.3    <none>        53/UDP,53/TCP,9153/TCP         20d   k8s-app=kube-dns   addonmanager.kubernetes.io/mode=Reconcile,k8s-app=kube-dns,kubernetes.io/name=coredns
```

The effect is as follows:

```shell
Chain EDGE-MESH (2 references)
 pkts bytes target     prot opt in     out     source               destination
    0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.0.3           /* ignore kube-system/coredns service */
    0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.0.1           /* ignore default/kubernetes service */
    7   420 EDGE-MESH-TCP  tcp  --  *      *       0.0.0.0/0            10.10.0.0/16         /* tcp service proxy */
```

**Customize the ignore method of other services**

You may also have some other services that you don't want to be proxied by edgemesh-agent. We have added an optional label for these services: `label: noproxy=edgemesh`, and it is recommended to configure this label for these services.

For example: if you want to deploy prometheus + grafana monitoring services in the cloud, these services do not require edge access, so you don't want them to be proxied by edgemesh-agent, then you can do:

1. Edit node-exporter-service.yaml

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    noproxy: edgemesh # add this label to ignored by edgemesh-agent
    app.kubernetes.io/name: node-exporter
    app.kubernetes.io/version: v1.0.1
  name: node-exporter
  namespace: monitoring
spec:
  clusterIP: None
  ports:
  - name: https
    port: 9100
    targetPort: https
  selector:
    app.kubernetes.io/name: node-exporter
```

2. Set the label similarly for other services

```shell
$ grep noproxy *
alertmanager-service.yaml:    noproxy: edgemesh
grafana-service.yaml:    noproxy: edgemesh
kube-state-metrics-service.yaml:    noproxy: edgemesh
node-exporter-service.yaml:    noproxy: edgemesh
prometheus-adapter-service.yaml:    noproxy: edgemesh
prometheus-service.yaml:    noproxy: edgemesh
```

The effect is as follows, these related services will be set to `RETURN` to return to the next chain of `EDGE-MESH`, namely `KUBE-SERVICES` for traffic forwarding:

```shell
Chain EDGE-MESH (4 references)
num   pkts bytes target     prot opt in     out     source               destination
1        1    60 RETURN     all  --  *      *       0.0.0.0/0            10.244.3.58          /* ignore monitoring/prometheus-operator service */
2    89726 5384K RETURN     all  --  *      *       0.0.0.0/0            10.10.120.87         /* ignore monitoring/prometheus-k8s service */
3        0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.250.210        /* ignore monitoring/grafana service */
4        0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.212.0          /* ignore monitoring/prometheus-adapter service */
5    21481 2496K RETURN     all  --  *      *       0.0.0.0/0            192.168.132.38       /* ignore monitoring/node-exporter service */
6    42917 4863K RETURN     all  --  *      *       0.0.0.0/0            192.168.132.37       /* ignore monitoring/node-exporter service */
7    22626 2428K RETURN     all  --  *      *       0.0.0.0/0            192.168.132.26       /* ignore monitoring/node-exporter service */
8     370K   33M RETURN     all  --  *      *       0.0.0.0/0            192.168.132.21       /* ignore monitoring/node-exporter service */
9    37099 4060K RETURN     all  --  *      *       0.0.0.0/0            192.168.132.19       /* ignore monitoring/node-exporter service */
10       1    60 RETURN     all  --  *      *       0.0.0.0/0            10.244.3.61          /* ignore monitoring/kube-state-metrics service */
11       0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.161.16         /* ignore monitoring/alertmanager-main service */
12   45443 2726K RETURN     all  --  *      *       0.0.0.0/0            10.10.0.3            /* ignore kube-system/coredns service */
13    2953  177K RETURN     all  --  *      *       0.0.0.0/0            10.10.0.1            /* ignore default/kubernetes service */
14      62  3720 EDGE-MESH-TCP  tcp  --  *      *       0.0.0.0/0            10.10.0.0/16         /* tcp service proxy */
```
