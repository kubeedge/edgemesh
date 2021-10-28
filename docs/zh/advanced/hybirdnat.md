# 混合 NAT

## 目标

集群同时部署了 `kube-proxy` 和 `edgemesh` 时，通过下面的方式保证了两者的兼容：

当 `edgemesh-agent` 后于 `kube-proxy` 启动的时候，会将自定义的 `EDGE-MESH` 链插入到主机 iptables 的 nat 表中的 `PREROUTING` 和 `OUTPUT` 链上，进而可以优先于 `kube-proxy` 劫持服务的流量，实现云边通信的流量转发。

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


## 流量代理

`edgemesh-agent` 默认拦截并代理转发所有通过 Cluster IP 访问的服务，不过 `edgemesh-agent` 的流量转发过程在用户态中，会带来些许的性能损耗。为提高流量转发效率，我们支持对服务的流量代理进行忽略设置，使其采用 `kube-proxy` 的代理进行流量转发。

**默认被忽略的服务**

目前有两个集群服务是默认被忽略的，分别是 kube-system 命名空间下的 kube-dns(coredns) 服务，以及 default 命名空间下的 kubernetes 服务。

以 kube-dns 服务为例子：程序会根据服务存在 `label: k8s-app=kube-dns` 的 kube-dns 服务来为其进行设置忽略，使其可以绕过 `PREROUTING -> EDGE-MESH`，再 `RETURN` 回到 kube-proxy 设置的 `KUBE-SERVICES` 链，继续由内核的 netfilter 进行流量转发。

```shell
$ kubectl get svc -owide -nkube-system --show-labels|grep coredns
coredns   ClusterIP   10.10.0.3    <none>        53/UDP,53/TCP,9153/TCP         20d   k8s-app=kube-dns   addonmanager.kubernetes.io/mode=Reconcile,k8s-app=kube-dns,kubernetes.io/name=coredns
```

效果如下:

```shell
Chain EDGE-MESH (2 references)
 pkts bytes target     prot opt in     out     source               destination
    0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.0.3           /* ignore kube-system/coredns service */
    0     0 RETURN     all  --  *      *       0.0.0.0/0            10.10.0.1           /* ignore default/kubernetes service */
    7   420 EDGE-MESH-TCP  tcp  --  *      *       0.0.0.0/0            10.10.0.0/16         /* tcp service proxy */
```

**自定义其他服务的忽略方法**

你可能还会有一些其他的服务不想被 edgemesh-agent 代理，我们为这些服务增加了一个可选项的标签: `label: noproxy=edgemesh`，并建议为这些服务配置该标签。

举个例子：比如你想在云端部署 prometheus + grafana 监控服务，这些服务不需要边缘端的访问，所以你并不希望它们被 edgemesh-agent 代理，那么你可以这么做：

1. 编辑 node-exporter-service.yaml

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

2. 类似的为其他服务也设置该标签

```shell
$ grep noproxy *
alertmanager-service.yaml:    noproxy: edgemesh
grafana-service.yaml:    noproxy: edgemesh
kube-state-metrics-service.yaml:    noproxy: edgemesh
node-exporter-service.yaml:    noproxy: edgemesh
prometheus-adapter-service.yaml:    noproxy: edgemesh
prometheus-service.yaml:    noproxy: edgemesh
```

效果如下，相关这些服务将被设置为 `RETURN` 回到 `EDGE-MESH` 的下一条链，即 `KUBE-SERVICES` 来进行流量转发:

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
