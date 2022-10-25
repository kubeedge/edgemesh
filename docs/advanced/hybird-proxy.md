# Hybrid Proxy

When kube-proxy and edgemesh are deployed in a K8s cluster at the same time, the compatibility of the two is ensured by hybrid proxy.

## Principle

When edgemesh-agent is started after kube-proxy, the `KUBE-PORTALS-CONTAINER` chain will be inserted into the PREROUTING and OUTPUT chains in the nat table of the host iptables, and it will be in front of the `KUBE-SERVICES` chain, so It can hijack the traffic of the service in preference to kube-proxy, and realize the traffic forwarding of cloud edge communication.

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
If there is an error such as `No route to host` when accessing the service, it may be because the startup of edgemesh-agent precedes the startup of kube-proxy, resulting in the chain of `KUBE-PORTALS-CONTAINER` located behind `KUBE-SERVICES`, which cannot be prioritized Because kube-proxy hijacks the service's traffic. This problem can be circumvented by redeploying edgemesh to re-establish the order of the chain.
:::

## Service Filter

edgemesh-agent intercepts and forwards all services accessed through Cluster IP by default, but the traffic forwarding process of edgemesh-agent will bring a little performance loss in user mode. Therefore, you may have some services that you don't want to be proxied by edgemesh-agent, we provide a label for service filtering: `service.edgemesh.kubeedge.io/service-proxy-name`.

To enable more fine-tuned settings you can set the service filter mode by setting the `serviceFilterMode` option in the edgemesh-agent configuration file (`modules.edgeProxy.serviceFilterMode`) to either "FilterIfLabelExists" or "FilterIfLabelDoesNotExists". The default value is "FilterIfLabelExists". If the value is "FilterIfLabelExists", the service will be filtered if the label `service.edgemesh.kubeedge.io/service-proxy-name` exists and vice versa if the label does not exist.

For example: If you want to deploy prometheus + grafana monitoring services in the cloud, these services do not require edge access, so you do not want them to be proxied by edgemesh-agent, then you can do this:

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    service.edgemesh.kubeedge.io/service-proxy-name: ""  #<---- add this label to ignored by edgemesh-agent
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

After saving the configuration, you will not see the iptables rules for this service in the `KUBE-PORTALS-CONTAINER` chain, meaning edgemesh will not process this service.
