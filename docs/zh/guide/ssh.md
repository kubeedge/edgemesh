# SSH 代理隧道

EdgeMesh 的 SSH 代理提供了节点之间通过代理进行 SSH 登录访问的能力，本章节会对此功能进行详细介绍。

## SSH 代理工作原理

![edgemesh-socks5-proxy](/images/guide/em-sock5.png)

1. 客户端通过代理发起远程登录请求，流量将会被转发到代理服务中
2. 在代理服务器中解析目的主机名和端口将其转换成远程服务器 IP 地址
3. 通过原有 Tunnel 模块的 P2P 打洞功能将流量转发至目标机器
4. 通道建立后对远端流量的响应返回给 SSH 客户端，完成通道建立

## 配置

### Helm 配置

通过 Helm 的 `--set` 参数：

```shell
$ helm install edgemesh --namespace kubeedge \
--set agent.modules.edgeProxy.socks5Proxy.enable=true ...
```

### 手动配置

```shell
$ vim build/agent/resources/04-configmap.yaml
  modules:
    ...
    edgeProxy:
      ...
      socks5Proxy:
        enable: true
    ...
```

::: warning
更改后需要重新启动 edgemesh-agent。
:::

## 使用

**由于节点的 IP 可能重复，所以只支持通过节点名称进行连接**

```shell
$ kubectl get node
NAME                                  STATUS   ROLES                         AGE    VERSION
master                                Ready    control-plane,master,worker   22d    v1.21.5
edge-node-1002                        Ready    agent,edge                    2d4h   v1.19.3-kubeedge-v1.7.2


$ ssh -o "ProxyCommand nc --proxy-type socks5 --proxy 169.254.96.16:10800 %h %p" root@edge-node-1002

ECDSA key fingerprint is SHA256:uPUzjIPK+zvu8ymzrkd0IWSsmrs2r/Hl72iendYniVY.
ECDSA key fingerprint is MD5:ca:80:85:84:bd:09:a6:fd:d9:ba:73:1c:b5:7c:f6:ae.
Are you sure you want to continue connecting (yes/no)? yes
root@edge-node-1002's password:

Activate the web console with: systemctl enable --now cockpit.socket

Last failed login: Wed Dec 15 14:45:06 CST 2021 from 192.168.1.128 on ssh:notty
There was 1 failed login attempt since the last successful login.
Last login: Wed Dec 15 14:07:04 2021 from 192.168.1.128
[root@edge-node-1002 ~]#
```

::: tip
不同的代理工具参数会不一样，具体请查看对应工具的文档。常用的代理工具有 nc, ncat
:::
