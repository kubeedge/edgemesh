# Developer Guide

There is a `Makefile` in the root folder.The common commands used by developers are as follows:

Build all binaries (`edgemesh-agent`, `edgemesh-server`)
```bash
make all
```

Build `edgemesh-agent` binary
```bash
make all WHAT=edgemesh-agent
```

Build `edgemesh-server` binary
```bash
make all WHAT=edgemesh-server
```

Build all images (`edgemesh-agent`, `edgemesh-server`)
```bash
make images
```

Build edgemesh-server image
```bash
make serverimage
```

Build edgemesh-agent image
```bash
make agentimage
```
Build all docker images for specific architecture.
```bash
make release ARCH=arm64
```

Build a new kubeedge cluster to test EdgeMesh's e2e test.
```bash
make e2e
```

Do lint
```bash
make lint
```
Run local verifications
```bash
make verify
```


Cross Build edgemesh-agent and edgemesh-server image
```shell
make docker-cross-build
```