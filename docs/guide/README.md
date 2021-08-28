# Introduction

As the data plane component of the [KubeEdge](https://github.com/kubeedge/kubeedge) cluster, EdgeMesh uses LibP2P technology to offer sample capacities (e.g, service discovery, traffic proxy, etc.) for applications running on the KubeEdge cluster, thus shielding the complex network topology at the edge scenairo.

## Background

KubeEdge is build based on [Kubernetes](https://github.com/kubernetes/kubernetes), extending cloud-native containerized application orchestration capabilities to the edge. However, at the scenario of edge computer, the network topology is more complex. Edge nodes in different areas are often not interconnected, and the inter-communication of traffic between applications is the primary requirement of the business. For this scenairo, EdgeMesh offers a solution.

## Why EdgeMesh?
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

## Key Features

<table align="center">
	<tr>
		<th align="center">Feature</th>
		<th align="center">Sub-Feature</th>
		<th align="center">Realization Degree</th>
	</tr >
	<tr >
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
		<td align="center">External Access</td>
		<td align="center">/</td>
		<td align="center">✓</td>
	</tr>
	<tr>
		<td align="center">Multi-NIC Monitoring</td>
		<td align="center">/</td>
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
