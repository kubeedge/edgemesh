# 混合 NAT

## 目标

集群同时部署了 `kube-proxy` 和 `edgemesh` 时，通过下面的方式保证了两者的兼容：

当 `edgemesh-agent` 后于 `kube-proxy` 启动的时候，会将自定义的 `EDGEMESH-` 链插入到主机 iptables 的 nat 表中的 `PREROUTING` 和 `OUTPUT` 链上，进而可以优先于 `kube-proxy` 劫持服务的流量，实现云边通信的流量转发。

```shell
Chain PREROUTING (policy ACCEPT 1379K packets, 395M bytes)
 pkts bytes target     prot opt in     out     source               destination
   52 13980 EDGEMESH-PORTALS-CONTAINER  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* handle ClusterIPs; NOTE: this must be before the NodePort rules */
1375K  394M KUBE-SERVICES  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* kubernetes service portals */

Chain OUTPUT (policy ACCEPT 1317K packets, 79M bytes)
 pkts bytes target     prot opt in     out     source               destination
   51  3086 EDGEMESH-PORTALS-HOST  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* handle ClusterIPs; NOTE: this must be before the NodePort rules */
1314K   79M KUBE-SERVICES  all  --  *      *       0.0.0.0/0            0.0.0.0/0            /* kubernetes service portals */
```

## 服务过滤

`edgemesh-agent` 默认拦截并转发所有通过 Cluster IP 访问的服务，不过 `edgemesh-agent` 的流量转发过程在用户态中，会带来些许的性能损耗。因此，你可能会有一些服务不想被 edgemesh-agent 代理，我们为服务过滤提供了一个标签: `service.edgemesh.kubeedge.io/service-proxy-name`。

举个例子：比如你想在云端部署 prometheus + grafana 监控服务，这些服务不需要边缘端的访问，所以你并不希望它们被 edgemesh-agent 代理，那么你可以这么做：

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    service.edgemesh.kubeedge.io/service-proxy-name: "" # add this label to ignored by edgemesh-agent
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

保存配置后，你将不会在 `EDGEMESH-` 链里看到这个服务的 iptables 规则，意味着 edgemesh 不会对此服务进行处理。
