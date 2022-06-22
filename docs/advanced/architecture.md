# Architecture

## Overview

![edgemesh-architecture](../.vuepress/public/images/advanced/em-arch.png)

The above figure shows a brief overview of the EdgeMesh architecture, EdgeMesh contains edgemesh-server and edgemesh-agent.

The core components of edgemesh-server include:

- **Tunnel-Server**: Based on [LibP2P](https://github.com/libp2p/go-libp2p), establish a connection with edgemesh-agent to provide relay capability and hole punching capability

The core components of edgemesh-agent include:

- **Proxier**: Responsible for configuring the kernel's iptables rules, and intercepting requests to the EdgeMesh process
- **DNS**: Built-in DNS resolver, which resolves the DNS request in the node into a service cluster IP
- **Traffic**: A traffic forwarding module based on the Go-Chassis framework, which is responsible for forwarding traffic between applications
- **Controller**: Obtains metadata (e.g., Service, Endpoints, Pod, etc.) through the Local APIServer capability on the edge side of KubeEdge
- **Tunnel-Agent**: Based on LibP2P, using relay and hole punching to provide the ability of communicating across subnets

:::tip
To ensure the capability of service discovery in some edge devices with low-version kernels or low-version iptables, EdgeMesh adopts the userspace mode in its implementation of the traffic proxy.
:::

## How It Works

- Through the capability of Local APIServer on the edge of KubeEdge, EdgeMesh monitors the addition, deletion and modification of metadata (e.g., Services and Endpoints), and then maintain the metadata required to access the services. At the same time configure iptables rules to intercept requests for the Cluster IP network segment.
- EdgeMesh uses the same ways (e.g., Cluster IP, domain name) as the K8s Service to access services
- Suppose we have two services, APP-A and APP-B, and now the APP-A service tries to access APP-B based on the domain name, the domain name resolution request will be intercepted by the edgemesh-agent of the node and edgemesh-agent will return the Cluster IP. This request will be redirected by the iptables rules previously configured by edgemesh-agent and forwarded to the port 40001 which is occupied by the EdgeMesh process (data packet from kernel mode -> user mode)
- After the request enters the edgemesh-agent process, the edgemesh-agent process completes the selection of the backend Pod (load balancing occurs here), and then the request will be sent to the edgemesh-agent of the host where APP-B is located through the tunnel module (via relay forwarding or direct transmission through holes punch)
- The edgemesh-agent of the node where APP-B is located is responsible for forwarding traffic to the service port of APP-B, and get the response back to the edgemesh-agent where APP-A is located
- The edgemesh-agent of the node where APP-A is located is responsible for feeding back the response data to the APP-A service

## Future Works

![edgemesh-future-work](../.vuepress/public/images/advanced/future-work.png)

At present, the implementation of EdgeMesh relies on the connectivity of the host network. In the future, EdgeMesh will realize the capabilities of CNI plug-ins, and realize the Pod network connectivity between edge nodes and nodes on the cloud, or edge nodes across LANs in a  compatible manner with mainstream CNI plug-ins (e.g., flannel / calico, etc). Finally, EdgeMesh can even replace part of its own components with cloud-native components (e.g., replacing [kube-proxy](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/) to achieve the capabilities of the Cluster IP, replacing [node local dns cache ](https://kubernetes.io/docs/tasks/administer-cluster/nodelocaldns/) to achieve node-level dns capabilities, and replace [envoy](https://www.envoyproxy.io/) to achieve mesh-layer capabilities).
