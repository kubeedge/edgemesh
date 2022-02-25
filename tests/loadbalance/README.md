# EdgeMesh LoadBalance Test

## Build

```shell
$ go build -o loadbalance .
```

## Usage

```shell
$ ./loadbalance --help
Usage of ./loadbalance:
  -counter int
    	Test count (default 500)
  -deployment string
    	Deployment yaml file
  -destination-rule string
    	Destination rule yaml file
  -kube-config string
    	Use this key to set kube-config path, eg: $HOME/.kube/config (default "/root/.kube/config")
  -namespace string
    	All resources namespace (default "default")
  -service string
    	Service yaml file
```

## Test1: Round Robin

```shell
$ ./loadbalance -counter 10000 -deployment hostname-deploy.yaml -service hostname-svc.yaml

Failure: 0, Success: 10000
hostname-lb-test-bf9ccbc8b-l2pv6: 1861	(18.6%)
hostname-lb-test-bf9ccbc8b-bvvx7: 2061	(20.6%)
hostname-lb-test-bf9ccbc8b-6nxkl: 2014	(20.1%)
hostname-lb-test-bf9ccbc8b-k8lvx: 2029	(20.3%)
hostname-lb-test-bf9ccbc8b-nx5fj: 2035	(20.3%)
```

## Test2: Random

```shell
$ ./loadbalance -counter 10000 -deployment hostname-deploy.yaml -service hostname-svc.yaml -destination-rule hostname-dr-random.yaml

Failure: 0, Success: 10000
hostname-lb-test-bf9ccbc8b-d65n6: 2008	(20.1%)
hostname-lb-test-bf9ccbc8b-9kwz8: 1963	(19.6%)
hostname-lb-test-bf9ccbc8b-mq6g6: 2305	(23.1%)
hostname-lb-test-bf9ccbc8b-gr4dm: 1806	(18.1%)
hostname-lb-test-bf9ccbc8b-h8lwm: 1918	(19.2%)
```
