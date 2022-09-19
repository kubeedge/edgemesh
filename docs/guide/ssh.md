# SSH Proxy Tunnel

EdgeMesh's Socks5 agent provides the ability of SSH login and access between nodes through Socks5 agent. This chapter will introduce this function in detail.

## How SSH Agent Works

![edgemesh-socks5-proxy](/images/guide/em-sock5.png)

1. The client initiates a remote login request through the proxy, and the traffic will be forwarded to the proxy service
2. Resolve the destination host name and port in the proxy server to convert it to the remote server IP address
3. The traffic is forwarded to the target machine through the P2P hole punching function of the original Tunnel module
4. After the channel is established, the response to the remote traffic is returned to the SSH client to complete the channel establishment

## Configuration

### Helm Configuration

Via Helm's `--set` parameter:

```shell
$ helm install edgemesh --namespace kubeedge \
--set agent.modules.edgeProxy.socks5Proxy.enable=true ...
```

### Manual Configuration

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
edgemesh-agent needs to be restarted after changing the configuration.
:::

## How to Use

**Since the IP address of the node may be duplicate, it only supports connection through the node name**

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
The parameters of different proxy tools will be different, please check the documentation of the corresponding tool for details. Commonly used proxy tools are nc, ncat
:::
