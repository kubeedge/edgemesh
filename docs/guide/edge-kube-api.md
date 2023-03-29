# Edge Kube-API Endpoint

## Background

Kubernetes has greatly improved its scalability through the CRD and Controller mechanisms, enabling many applications to be easily integrated into the Kubernetes ecosystem. As we all know, most Kubernetes applications access kube-apiserver to obtain basic metadata, such as Service, Pod, Job, Deployment, etc., as well as CRD metadata based on their own business expansion.

However, in edge computing scenarios, due to the lack of network connectivity, edge nodes usually cannot directly connect to the kube-apiserver service on the cloud, so that Kubernetes applications deployed on the edge cannot obtain the metadata it needs. For example, Kube-Proxy and Flannel that are scheduled to edge nodes usually do not work properly.

**Fortunately, KubeEdge >= v1.7.0 provides the ability of Edge Kube-API Endpoint, which can provide services similar to kube-apiserver on the cloud, so that edge applications that require kube-apiserver can also be unaware run on the edge.** This chapter will guide you on how to configure KubeEdge to enable the Edge Kube-API Endpoint, and how EdgeMesh uses it.

## Quick Start 

- **Step 1**: On the cloud, enable the dynamicController module. After the configuration is complete, you need to restart cloudcore

```yaml
$ vim /etc/kubeedge/config/cloudcore.yaml
modules:
  ...
  dynamicController:
    enable: true
...
```
In case you have installed cloudcore using keadm, the configuration file will not be present. Instead, you can enable it using 
```
keadm init --advertise-address="THE-EXPOSED-IP" --profile version=v1.12.1 --kube-config=/root/.kube/config --set cloudCore.modules.dynamicController.enable=true
```


- **Step 2**: At the edge node, open the metaServer module (if your KubeEdge < 1.8.0, you need to close the old edgeMesh module). After the configuration is complete, you need to restart edgecore

```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edgeMesh:
    enable: false
  ...
  metaManager:
    metaServer:
      enable: true
...
```

- **Step 3**: At the edge node, configure clusterDNS and clusterDomain. After the configuration is complete, you need to restart edgecore

```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edged:
    clusterDNS: 169.254.96.16
    clusterDomain: cluster.local
...
```

If KubeEdge >= v1.12.0, configure it like this:
```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  edged:
    ...
    tailoredKubeletConfig:
      ...
      clusterDNS:
      - 169.254.96.16
      clusterDomain: cluster.local
...
```

::: tip
- The configuration in step 3 is for the edge application to be able to access the DNS service of EdgeMesh, and has nothing to do with the Edge Kube-API Endpoint itself, but for the fluency of configuration, it is still explained here.
- The value '169.254.96.16' set by clusterDNS comes from the default value of bridgeDeviceIP in [commonConfig](https://edgemesh.netlify.app/reference/config-items.html#edgemesh-agent-cfg). There is no need to modify it below. If you have to modify it, please keep the two consistent.
:::

- **Step 4**: Finally, at the edge node, test whether the Edge Kube-API Endpoint functions properly

```shell
$ curl 127.0.0.1:10550/api/v1/services
{"apiVersion":"v1","items":[{"apiVersion":"v1","kind":"Service","metadata":{"creationTimestamp":"2021-04-14T06:30:05Z","labels":{"component":"apiserver","provider":"kubernetes"},"name":"kubernetes","namespace":"default","resourceVersion":"147","selfLink":"default/services/kubernetes","uid":"55eeebea-08cf-4d1a-8b04-e85f8ae112a9"},"spec":{"clusterIP":"10.96.0.1","ports":[{"name":"https","port":443,"protocol":"TCP","targetPort":6443}],"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}},{"apiVersion":"v1","kind":"Service","metadata":{"annotations":{"prometheus.io/port":"9153","prometheus.io/scrape":"true"},"creationTimestamp":"2021-04-14T06:30:07Z","labels":{"k8s-app":"kube-dns","kubernetes.io/cluster-service":"true","kubernetes.io/name":"KubeDNS"},"name":"kube-dns","namespace":"kube-system","resourceVersion":"203","selfLink":"kube-system/services/kube-dns","uid":"c221ac20-cbfa-406b-812a-c44b9d82d6dc"},"spec":{"clusterIP":"10.96.0.10","ports":[{"name":"dns","port":53,"protocol":"UDP","targetPort":53},{"name":"dns-tcp","port":53,"protocol":"TCP","targetPort":53},{"name":"metrics","port":9153,"protocol":"TCP","targetPort":9153}],"selector":{"k8s-app":"kube-dns"},"sessionAffinity":"None","type":"ClusterIP"},"status":{"loadBalancer":{}}}],"kind":"ServiceList","metadata":{"resourceVersion":"377360","selfLink":"/api/v1/services"}}
```

::: warning
If the return value is an empty list, or the response takes a long time (close to 10s) to get the return value, your configuration may be wrong, please check carefully.
:::

**After completing the above steps, the Edge Kube-API Endpoint function of KubeEdge has been enabled, and then continue to deploy EdgeMesh.**

## Security

KubeEdge >= v1.12.0 [hardens the security](https://github.com/kubeedge/kubeedge/issues/4108) of the Edge Kube-API Endpoint to support HTTPS secure access. If you want to harden the security of the Edge Kube-API Endpoint service, this section will guide you how to configure KubeEdge to enable the secure Edge Kube-API Endpoint feature and how EdgeMesh uses it.

### Configure

- **Step 1**: Enable KubeEdge's requireAuthorization feature gate

Both cloudcore.yaml and edgecore.yaml are configured as follows. After the configuration is complete, you need to restart cloudcore and edgecore

```yaml
$ vim /etc/kubeedge/config/cloudcore.yaml
kind: CloudCore
featureGates:
  requireAuthorization: true
modules:
  ...
```

```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
kind: EdgeCore
featureGates:
  requireAuthorization: true
modules:
  ...
```

- **Step 2**: Generate self-signed certificate

We borrow KubeEdge's `certgen.sh` script to generate self-signed certificates. **Note, please do not do this in production environment, please use production-level certificate for production environment.**

```shell
# 1. Confirm that the /etc/kubernetes/pki/ directory exists
$ ls /etc/kubernetes/pki/

# 2. Create a directory
$ mkdir -p /tmp/metaserver-certs
$ cd /tmp/metaserver-certs

# 3. Download certgen.sh
$ wget https://raw.githubusercontent.com/kubeedge/kubeedge/master/build/tools/certgen.sh
$ chmod u+x certgen.sh

# 4. Generate certificate file
$ CA_PATH=./ CERT_PATH=./ ./certgen.sh stream

# 5. rename the certificate file
$ mv streamCA.crt rootCA.crt; mv stream.crt server.crt; mv stream.key server.key

# 6. Create certificate secret
$ kubectl -n kubeedge create secret generic metaserver-certs --from-file=./rootCA.crt --from-file=./server.crt --from-file=./server.key
```

- **Step 3**: At the edge node, configure the certificate path of the metaServer. After the configuration is complete, you need to restart edgecore

```yaml
$ vim /etc/kubeedge/config/edgecore.yaml
modules:
  ...
  metaManager:
    metaServer:
      enable: true
      server: https://127.0.0.1:10550
      tlsCaFile: /tmp/metaserver-certs/rootCA.crt
      tlsCertFile: /tmp/metaserver-certs/server.crt
      tlsPrivateKeyFile: /tmp/metaserver-certs/server.key
...
```

**After completing the above configuration and restarting, you can have an HTTPS-based, secure Edge Kube-API Endpoint service, you can refer to [issue#4801](https://github.com/kubeedge/kubeedge/issues/4108), and use `curl` to test if it works.**

### Usage

EdgeMesh has the following two ways to connect to the HTTPS-based Edge Kube-API Endpoint, you can choose one of them according to the actual situation.

#### Method 1: One-Way Authentication

EdgeMesh can access HTTPS-based Edge Kube-API Endpoint through One-Way authentication. One-Way authentication does not require client certificate verification.

- Helm Configure

```shell
helm install edgemesh --namespace kubeedge \
--set agent.kubeAPIConfig.metaServer.security.requireAuthorization=true \
--set agent.kubeAPIConfig.metaServer.security.insecureSkipTLSVerify=true \
...
```

- Manual Configuration

```yaml
$ vim build/agent/resources/04-configmap.yaml
...
data:
  edgemesh-agent.yaml: |
    kubeAPIConfig:
      metaServer:
        security:
          requireAuthorization: true
          insecureSkipTLSVerify: true
    ...
```

#### Method 2: Two-Way Authentication

EdgeMesh can also access HTTPS-based Edge Kube-API Endpoint through Two-Way authentication. The Two-Way authentication needs to verify both the server certificate and the client certificate.

- Helm Configure

```shell
helm install edgemesh --namespace kubeedge \
--set agent.kubeAPIConfig.metaServer.security.requireAuthorization=true \
--set agent.metaServerSecret=metaserver-certs \
...
```

- Manual Configuration

```yaml
$ vim build/agent/resources/04-configmap.yaml
...
data:
  edgemesh-agent.yaml: |
    kubeAPIConfig:
      metaServer:
        security:
          requireAuthorization: true
    ...
```

```yaml
$ vim build/agent/resources/05-daemonset.yaml
...
        volumeMounts:
        ...
        - name: metaserver-certs
          mountPath: /etc/edgemesh/metaserver
        volumes:
        ...
        - secret:
          secretName: metaserver-certs
          defaultMode: 420
        name: metaserver-certs
        ...
```
