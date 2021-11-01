English | [简体中文](./README_zh.md)

# EdgeMesh

[![CI](https://github.com/kubeedge/edgemesh/actions/workflows/main.yaml/badge.svg?branch=main)](https://github.com/kubeedge/edgemesh/actions/workflows/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/edgemesh)](https://goreportcard.com/report/github.com/kubeedge/edgemesh)
[![GitHub license](https://img.shields.io/github/license/kubeedge/edgemesh)](https://github.com/kubeedge/edgemesh/blob/main/LICENSE)


## Introduction

EdgeMesh, as the data plane component of the [KubeEdge](https://github.com/kubeedge/kubeedge) cluster, provides simple service discovery and traffic proxy functions for applications, thereby shielding the complex network structure in edge scenarios.

### Background

KubeEdge is build based on [Kubernetes](https://github.com/kubernetes/kubernetes), extending cloud-native containerized application orchestration capabilities to the edge. However, at the scenario of edge computer, the network topology is more complex. Edge nodes in different areas are often not interconnected, and the inter-communication of traffic between applications is the primary requirement of the business. For this scenairo, EdgeMesh offers a solution.

### Why EdgeMesh?
EdgeMesh satisfies the new requirements in edge scenarios (e.g., limited edge resources, unstable edge cloud network, complex network structure, etc.), that is, high availability, high reliability, and extreme lightweight:

- **High availability**
  - Use the capabilities provided by LibP2P to connect the network between edge nodes
  - Divide the communication between edge nodes into intra-LAN and cross-LAN
    - Intra-LAN communication: direct access
    - Cross-LAN communication: when the hole punching is successful, a connection channel is established between the Agents, otherwise it is forwarded through the Server relay
- **High reliability (offline scenario)**
  - Both control plane and data plane traffic are delivered through the edge cloud channel
  - EdgeMesh internally implements a lightweight DNS server, thus no longer accessing the cloud CoreDNS
- **Extreme lightweight**
  - Each node has one and only one Agent, which saves edge resources

**User value**

- Enable users to have the ability to access edge-to-edge/edge-to-cloud/cloud-to-edge applications across different LANs
- Compared to the mechanism of CoreDNS + Kube-Proxy + CNI service discovery, users only need to simply deploy an Agent to finish their goals

### Key Features

<table align="center">
	<tr>
		<th align="center">Feature</th>
		<th align="center">Sub-Feature</th>
		<th align="center">Realization Degree</th>
	</tr>
	<tr>
		<td align="center">Service Discovery</td>
		<td align="center">/</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td rowspan="4" align="center">Traffic Governance</td>
	 	<td align="center">HTTP</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">TCP</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">Websocket</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">HTTPS</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td rowspan="3" align="center">Load Balance</td>
	 	<td align="center">Random</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">Round Robin</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td align="center">Session Persistence</td>
		<td align="center">✓</td>
	</tr>
	<tr>
    <td rowspan="2" align="center">Edge Gateway</td>
    <td align="center">External Access</td>
    <td align="center">✓</td>
  </tr>
  <tr>
    <td align="center">Multi-NIC Monitoring</td>
    <td align="center">✓</td>
  </tr>
  <tr>
		<td rowspan="2" align="center">Cross-Subnet Communication</td>
	 	<td align="center">Cross-Cloud Communication</td>
		<td align="center">✓</td>
	</tr>
	<tr>
	 	<td align="center">Cross-LAN E2E Communication</td>
		<td align="center">✓</td>
	</tr>
  <tr>
		<td align="center">Edge CNI</td>
	 	<td align="center">Cross-Subnet Pod Communication</td>
		<td align="center">+</td>
	</tr>
</table>

**Noting:**

- `✓` Features supported by the EdgeMesh version
- `+` Features not available in the EdgeMesh version, but will be supported in subsequent versions
- `-` Features not available in the EdgeMesh version, or deprecated features


## Architecture

![image](./docs/.vuepress/public/images/advanced/em-arch.png)

The above figure shows a brief overview of the EdgeMesh architecture, EdgeMesh contains EdgeMesh-Server and EdgeMesh-Agent.

The core components of EdgeMesh-Server include:

- **Tunnel-Server**: Based on [LibP2P](https://github.com/libp2p/go-libp2p), establish a connection with EdgeMesh-Agent to provide relay capability and hole punching capability

The core components of EdgeMesh-Agent include:

- **Proxier**: Responsible for configuring the kernel's iptables rules, and intercepting requests to the EdgeMesh process
- **DNS**: Built-in DNS resolver, which resolves the DNS request in the node into a service cluster IP
- **Traffic**: A traffic forwarding module based on the Go-Chassis framework, which is responsible for forwarding traffic between applications
- **Controller**: Obtains metadata (e.g., Service, Endpoints, Pod, etc.) through the List-Watch capability on the edge side of KubeEdge
- **Tunnel-Agent**: Based on LibP2P, using relay and hole punching to provide the ability of communicating across subnets


## Guides

### Prerequisites
Before using EdgeMesh, you need to understand the following prerequisites at first:

- while using DestinationRule, the name of the DestinationRule must be equal to the name of the corresponding Service. EdgeMesh will determine the DestinationRule in the same namespace according to the name of the Service
- Service ports must be named. The key/value pairs of port name must have the following syntax: name: \<protocol>[-\<suffix>]

### Documents
Documentation is located on [netlify.com](https://edgemesh.netlify.app/). These documents can help you understand EdgeMesh better.

### Installation
Follow the [EdgeMesh installation document](https://edgemesh.netlify.app/guide/getting-started.html) to install EdgeMesh.

### Examples
Example1: [HTTP traffic forwarding](https://edgemesh.netlify.app/guide/test-case.html#http)

Example2: [TCP traffic forwarding](https://edgemesh.netlify.app/guide/test-case.html#tcp)

Example3: [Websocket traffic forwarding](https://edgemesh.netlify.app/guide/test-case.html#websocket)

Example4: [Load Balance](https://edgemesh.netlify.app/guide/test-case.html#load-balance)

Example5: [Cross-edge-cloud communication](https://edgemesh.netlify.app/guide/test-case.html#cross-edge-cloud)


## Contact

If you need support, start with the 'Operation Guidance', and then follow the process that we've outlined

If you have any question, please contact us through the recommended information on [KubeEdge](https://github.com/kubeedge/kubeedge#contact)


## Contributing
If you are interested in EdgeMesh and would like to contribute to EdgeMesh project, please refer to [CONTRIBUTING](./CONTRIBUTING.md) for detailed contribution process guide.
