# EdgeMesh Security

EdgeMesh has high security. First, the communication between edgemesh-agent (including edgemesh-gateway) is encrypted transmission by default, and the PSK mechanism is used to ensure identity authentication and connection access. The PSK mechanism ensures that each edgemesh-agent (including edgemesh-gateway) can only establish a connection if it has the same "PSK cipher".

## Generate PSK cipher

To generate a PSK cipher, you can use the following command to generate a random string to use as the PSK cipher, or you can customize a string to use as the PSK cipher.

```shell
$ openssl rand -base64 32
JugH9HP1XBouyO5pWGeZa8LtipDURrf17EJvUHcJGuQ=
```

:::warning
Do not use the PSK cipher above directly, it will make the cluster unreliable. At the same time, it is recommended to change the PSK cipher frequently to ensure the high security of the cluster.
:::

## Use PSK cipher

### Helm configuration

When deploying EdgeMesh or EdgeMesh-Gateway through Helm, you can use the `--set` parameter to configure your own PSK cipher:

```shell
# When deploying EdgeMesh
$ helm install edgemesh --namespace kubeedge --set agent.psk=<your psk cipher> ...

# When deploying EdgeMesh-Gateway
$ helm install edgemesh-gateway --namespace kubeedge --set psk=<your psk cipher> ...
```

:::warning
EdgeMesh and EdgeMesh-Gateway in the same cluster need to use the same PSK cipher.
:::

### Manual configuration

When manually deploying EdgeMesh, you can directly edit the psk value in build/agent/resources/04-configmap.yaml.

When manually deploying EdgeMesh-Gateway, you can directly edit the psk value in build/gateway/resources/04-configmap.yaml.
