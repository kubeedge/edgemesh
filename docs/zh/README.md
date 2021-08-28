---
home: true
title: 首页
heroImage: /images/hero.png
actions:
  - text: 快速上手
    link: /zh/guide/getting-started.html
    type: primary
  - text: 项目简介
    link: /zh/guide/
    type: secondary
features:
  - title: 跨子网通讯
    details: 基于 LibP2P 实现，利用中继和打洞技术来提供容器间的跨子网边边和边云通讯能力。
  - title: 云原生体验
    details: 为 KubeEdge 集群中的容器应用提供与云原生一致的服务发现与流量转发体验。
  - title: 轻量化
    details: 每个节点仅需部署一个 Agent，边缘侧无需依赖 CoreDNS、Kube-Proxy 和 CNI 插件等原生组件。
  - title: 低时延
    details: 通过 UDP 打洞，完成 Agent 之间的点对点直连，数据通信无需经过多次中转。
  - title: 高可靠性
    details: 在底层网络拓扑结构不支持打洞时，通过 Server 中继转发流量，保障服务之间的正常通讯。
  - title: 非侵入式设计
    details: 使用 Kubernetes Service 原生接口，无需自定义 CRD，降低用户学习和使用成本。
footer: 2021 © KubeEdge Project Authors. All rights reserved.
---
