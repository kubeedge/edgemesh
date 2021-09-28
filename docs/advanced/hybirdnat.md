# Hybrid-NAT

## Goal

When `kube-proxy` and `edgemesh` are configured in the cloud, it is necessary to consider how the two are compatible.

In order to ensure cloud-side communication, when the `edgemesh-agent-cloud` container is started in the cloud, `EDGE-MESH` will be inserted into the root chain of the `PREROUTING` and `OUTPUT` tables, which can hijack the traffic and realize the traffic forwarding of cloud-side communication.

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

## Ignore kube-dns or core-dns

Edgemesh's current traffic forwarding process is still in user mode, so in order to improve the efficiency of traffic forwarding, we have ignored the `kube-dns` or `core-dns` traffic forwarding settings in the cloud, so that it still uses the kernel mode for traffic forwarding.

We set the ignore according to the service name of `kube-dns` or the existence of the coredns service with `label: k8s-app=kube-dns`, so that it can bypass `PREROUTING -> EDGE-MESH` and make it `RETURN` returns to the `KUBE-SERVICES` chain set by kube-proxy, and continues traffic forwarding by the kernel's netfilter.

```shell
[root@dke-master1 manifests]# kubectl get svc -owide -nkube-system --show-labels|grep coredns
coredns   ClusterIP   10.10.0.3    <none>        53/UDP,53/TCP,9153/TCP         20d   k8s-app=kube-dns   addonmanager.kubernetes.io/mode=Reconcile,k8s-app=kube-dns,kubernetes.io/name=coredns
```

The effect is as follows:

```shell
Chain EDGE-MESH (4 references)
num   pkts bytes target     prot opt in     out     source               destination         
1   45443 2726K RETURN     all  --  *      *       0.0.0.0/0            10.10.0.3            /* ignore coredns.kube-system service */
2    2953  177K RETURN     all  --  *      *       0.0.0.0/0            10.10.0.1            /* ignore kubernetes.default service */
3      62  3720 EDGE-MESH-TCP  tcp  --  *      *       0.0.0.0/0            10.10.0.0/16         /* tcp service proxy */

```

## Other service ignore methods

There may be other services in the cloud that do not require edge access. To ensure the efficiency of traffic forwarding and reduce the workload pressure of `edgemesh-agent-cloud`, we have added an optional label to these services: `label: noproxy =edgemesh`, and it is recommended to configure this tag for these services.

### Example

To deploy prometheus+grafana monitoring services in the cloud, they do not require edge access:

node-exporter-service.yaml

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    noproxy: edgemesh         # add this label to ignored by edgemesh-agent-cloud
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

Similarly, set this label for other services:

```shell
[root@dke-master1 manifests]# grep noproxy *
alertmanager-service.yaml:    noproxy: edgemesh
grafana-service.yaml:    noproxy: edgemesh
kube-state-metrics-service.yaml:    noproxy: edgemesh
node-exporter-service.yaml:    noproxy: edgemesh
prometheus-adapter-service.yaml:    noproxy: edgemesh
prometheus-service.yaml:    noproxy: edgemesh
```

The effect is as follows, these related services will be set to `RETURN` to the next chain of `EDGE-MESH`, that is, `KUBE-SERVICES` for traffic forwarding:

```shell
Chain EDGE-MESH (4 references)
num   pkts bytes target     prot opt in     out     source               destination         
1        1    60 RETURN     all  --  *      *       0.0.0.0/0            10.244.3.58          /* ignore prometheus-operator.monitoring service */
2    89726 5384K RETURN     all  --  *      *       0.0.0.0/0            10.10.120.87         /* ignore prometheus-k8s.monitoring service */
3        0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.250.210        /* ignore grafana.monitoring service */
4        0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.212.0          /* ignore prometheus-adapter.monitoring service */
5    21481 2496K RETURN     all  --  *      *       0.0.0.0/0            192.168.132.38       /* ignore node-exporter.monitoring service */
6    42917 4863K RETURN     all  --  *      *       0.0.0.0/0            192.168.132.37       /* ignore node-exporter.monitoring service */
7    22626 2428K RETURN     all  --  *      *       0.0.0.0/0            192.168.132.26       /* ignore node-exporter.monitoring service */
8     370K   33M RETURN     all  --  *      *       0.0.0.0/0            192.168.132.21       /* ignore node-exporter.monitoring service */
9    37099 4060K RETURN     all  --  *      *       0.0.0.0/0            192.168.132.19       /* ignore node-exporter.monitoring service */
10       1    60 RETURN     all  --  *      *       0.0.0.0/0            10.244.3.61          /* ignore kube-state-metrics.monitoring service */
11       0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.161.16         /* ignore alertmanager-main.monitoring service */
12   45443 2726K RETURN     all  --  *      *       0.0.0.0/0            10.10.0.3            /* ignore coredns.kube-system service */
13    2953  177K RETURN     all  --  *      *       0.0.0.0/0            10.10.0.1            /* ignore kubernetes.default service */
14      62  3720 EDGE-MESH-TCP  tcp  --  *      *       0.0.0.0/0            10.10.0.0/16         /* tcp service proxy */
```
