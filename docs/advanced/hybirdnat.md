# Hybrid-NAT

## Goal

When both `kube-proxy` and `edgemesh` are deployed in the cluster, the compatibility of the two is ensured by the following methods:

When `edgemesh-agent` is started after `kube-proxy`, the custom `EDGEMESH-` chain will be inserted into the `PREROUTING` and `OUTPUT` chains in the nat table of the host iptables, which can take precedence over `kube-proxy` hijacks the traffic of the service and realizes the traffic forwarding of cloud-edge communication.

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

## Service Filtering

`edgemesh-agent` intercepts and forwards all services accessed through Cluster IP by default, but the traffic forwarding process of `edgemesh-agent` will bring a little performance loss in user mode. Therefore, you may have some services that you don't want to be proxied by edgemesh-agent, we provide a label for service filtering: `service.edgemesh.kubeedge.io/service-proxy-name`.

For example: If you want to deploy prometheus + grafana monitoring services in the cloud, these services do not require edge access, so you do not want them to be proxied by edgemesh-agent, then you can do this:

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    service.edgemesh.kubeedge.io/service-proxy-name: edgemesh # add this label to ignored by edgemesh-agent
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

After saving the configuration, you will not see the iptables rules for this service in the `EDGEMESH-` chain, meaning edgemesh will not process this service.
