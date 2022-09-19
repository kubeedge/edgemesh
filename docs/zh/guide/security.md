# EdgeMesh 安全配置

EdgeMesh 具备很高的安全性，首先 edgemesh-agent （包括 edgemesh-gateway）之间的通讯默认是加密传输的，同时通过 PSK 机制保障身份认证与连接准入。PSK 机制确保每个 edgemesh-agent （包括 edgemesh-gateway）只有当拥有相同的 “PSK 密码” 时才能建立连接。

## 生成 PSK 密码

生成 PSK 密码，可以通过下方命令生成一个随机字符串用作 PSK 密码，你也可以自定义一个字符串用作 PSK 密码。

```shell
$ openssl rand -base64 32
JugH9HP1XBouyO5pWGeZa8LtipDURrf17EJvUHcJGuQ=
```

:::warning
请勿直接使用上方的 PSK 密码，这会导致集群不可靠。同时建议经常更换 PSK 密码，以保障集群的高安全性。
:::

## 使用 PSK 密码

### Helm 配置

通过 Helm 部署 EdgeMesh 或 EdgeMesh-Gateway 时，可以使用 `--set` 参数，配置你自己的 PSK 密码：

```shell
# 部署 EdgeMesh 时
$ helm install edgemesh --namespace kubeedge --set agent.psk=<你的 PSK 密码> ...

# 部署 EdgeMesh-Gateway 时
$ helm install edgemesh-gateway --namespace kubeedge --set psk=<你的 PSK 密码> ...
```

:::warning
同一个集群里的 EdgeMesh 和 EdgeMesh-Gateway 需要使用相同的 PSK 密码。
:::

### 手动配置

手动部署 EdgeMesh 时，直接编辑 build/agent/resources/04-configmap.yaml 里的 psk 值即可。

手动部署 EdgeMesh-Gateway 时，直接编辑 build/gateway/resources/04-configmap.yaml 里的 psk 值即可。
