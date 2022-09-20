# 混合代理

K8s 集群同时部署了 kube-proxy 和 edgemesh 时，通过混合代理的方式保证了两者的兼容。

## 原理

当 edgemesh-agent 后于 kube-proxy 启动的时候，会将 `KUBE-PORTALS-CONTAINER` 链插入到主机 iptables 的 nat 表中的 PREROUTING 和 OUTPUT 链上，并处于 `KUBE-SERVICES` 链的前面，因此可以优先于 kube-proxy 劫持服务的流量，实现云边通信的流量转发。

```shell
$ iptables -t nat -nvL
Chain PREROUTING (policy ACCEPT 1379K packets, 395M bytes)
 pkts bytes target     prot opt in  out     source               destination
   52 13980 KUBE-PORTALS-CONTAINER  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* handle ClusterIPs; NOTE: this must be before the NodePort rules */
1375K  394M KUBE-SERVICES  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* kubernetes service portals */
...

Chain OUTPUT (policy ACCEPT 1317K packets, 79M bytes)
 pkts bytes target     prot opt in     out     source               destination
   51  3086 KUBE-PORTALS-HOST  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* handle ClusterIPs; NOTE: this must be before the NodePort rules */
1314K   79M KUBE-SERVICES  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* kubernetes service portals */
...
```

:::tip
如果访问服务出现如 `No route to host` 的错误，可能是因为 edgemesh-agent 的启动先于 kube-proxy 的启动，导致 `KUBE-PORTALS-CONTAINER` 链位于 `KUBE-SERVICES` 后面，进而无法优先于 kube-proxy 劫持服务的流量。可以通过重新部署 edgemesh 来重建链的顺序，以规避这个问题。
:::

## 服务过滤

edgemesh-agent 默认拦截并转发所有通过 Cluster IP 访问的服务，不过 edgemesh-agent 的流量转发过程在用户态中，会带来些许的性能损耗。因此，你可能会有一些服务不想被 edgemesh-agent 代理，我们为服务过滤提供了一个标签: `service.edgemesh.kubeedge.io/service-proxy-name`。

举个例子：比如你想在云端部署 prometheus + grafana 监控服务，这些服务不需要边缘端的访问，所以你并不希望它们被 edgemesh-agent 代理，那么你可以这么做：

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    service.edgemesh.kubeedge.io/service-proxy-name: ""  #<---- 添加此 label 以让 edgemesh-agent 忽略代理
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

保存配置后，你将不会在 `KUBE-PORTALS-CONTAINER` 链里看到这个服务的 iptables 规则，意味着 edgemesh 不会对此服务进行处理。
