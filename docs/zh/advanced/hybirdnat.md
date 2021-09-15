# 云端混合NAT

## 目标

在云端配置了`kube-proxy`和`edgemesh`的情况, 需要考虑两者如何兼容.

为保证云边通信, `edgemesh-agent-cloud`容器在云端启动的时候, 会将`EDGE-MESH`插入到`PREROUTING`和`OUTPUT`表的根链上, 从而可以劫持流量, 实现云边通信的流量转发.

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

## 忽略kube-dns或者core-dns

edgemesh目前的流量转发过程还在用户态中, 所以为提高流量转发效率, 我们将`云端`的kube-dns或者core-dns的流量转发进行了忽略设置, 即使其依然采用内核态进行流量转发.

我们根据服务名称为`kube-dns`或者存在`label: k8s-app=kube-dns`的coredns服务, 来为其进行设置忽略, 使其可以绕过`PREROUTING -> EDGE-MESH`, 使其`RETURN`回到kube-proxy设置的`KUBE-SERVICES`链, 继续由内核的netfilter进行流量转发.

```shell
[root@dke-master1 manifests]# kubectl get svc -owide -nkube-system --show-labels|grep coredns
coredns   ClusterIP   10.10.0.3    <none>        53/UDP,53/TCP,9153/TCP         20d   k8s-app=kube-dns   addonmanager.kubernetes.io/mode=Reconcile,k8s-app=kube-dns,kubernetes.io/name=coredns
```

效果如下:

```shell
Chain EDGE-MESH (4 references)
num   pkts bytes target     prot opt in     out     source               destination         
1   45443 2726K RETURN     all  --  *      *       0.0.0.0/0            10.10.0.3            /* ignore coredns.kube-system service */
2    2953  177K RETURN     all  --  *      *       0.0.0.0/0            10.10.0.1            /* ignore kubernetes.default service */
3      62  3720 EDGE-MESH-TCP  tcp  --  *      *       0.0.0.0/0            10.10.0.0/16         /* tcp service proxy */

```

## 其他服务忽略方法

 在云端可能还会有一些其他的服务, 不需要边缘访问, 为保证其流量转发效率, 并降低edgemesh-agent-cloud的负载压力, 我们为这些服务增加了一个可选项的标签: `label: noproxy=edgemesh`, 并建议为这些服务配置该标签.

### 例子:

以在云端部署prometheus+grafana监控服务, 这些服务不需要边缘端的访问:

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

类似的为其他服务也设置该标签:

```shell
[root@dke-master1 manifests]# grep noproxy *
alertmanager-service.yaml:    noproxy: edgemesh
grafana-service.yaml:    noproxy: edgemesh
kube-state-metrics-service.yaml:    noproxy: edgemesh
node-exporter-service.yaml:    noproxy: edgemesh
prometheus-adapter-service.yaml:    noproxy: edgemesh
prometheus-service.yaml:    noproxy: edgemesh
```

效果如下, 相关这些服务将被设置为`RETURN`回到`EDGE-MESH`的下一条链, 即`KUBE-SERVICES`来进行流量转发:

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
