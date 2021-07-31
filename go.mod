module github.com/kubeedge/edgemesh

go 1.14

require (
	github.com/buraksezer/consistent v0.0.0-20191006190839-693edf70fd72
	github.com/cespare/xxhash v1.1.0
	github.com/go-chassis/go-archaius v0.20.0
	github.com/go-chassis/go-chassis v1.7.1
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.3 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/kubeedge/beehive v0.0.0
	github.com/kubeedge/kubeedge v1.6.2
	github.com/libp2p/go-libp2p v0.13.1-0.20210224102305-f981b25d2738
	github.com/libp2p/go-libp2p-circuit v0.4.0
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-msgio v0.0.6
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/vishvananda/netlink v1.1.0
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20210423184538-5f58ad60dda6 // indirect
	golang.org/x/sys v0.0.0-20210426080607-c94f62235c83 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	istio.io/api v0.0.0-20210131044048-bfeb10697307
	istio.io/client-go v0.0.0-20210218000043-b598dd019200
	k8s.io/api v0.19.10
	k8s.io/apimachinery v0.19.10
	k8s.io/client-go v0.19.10
	k8s.io/component-base v0.19.10
	k8s.io/klog/v2 v2.2.0
	k8s.io/kubernetes v1.19.10
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/kubeedge/beehive v0.0.0 => github.com/kubeedge/beehive v0.0.0-20201125122335-cd19bca6e436
	github.com/kubeedge/viaduct v0.0.0 => github.com/kubeedge/viaduct v0.0.0-20210601015050-d832643a3d35
	k8s.io/api v0.0.0 => k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.19.3
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.19.3
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.19.3
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.19.3
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.19.3
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.19.3
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.19.3
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.19.3
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.19.3
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.19.3
	k8s.io/csi-api v0.0.0 => k8s.io/csi-api v0.19.3
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.19.3
	k8s.io/gengo v0.0.0 => k8s.io/gengo v0.19.3
	k8s.io/heapster => k8s.io/heapster v1.2.0-beta.1 // indirect
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.2.0
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.19.3
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.19.3
	k8s.io/kube-openapi v0.0.0 => k8s.io/kube-openapi v0.19.3
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.19.3
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.19.3
	k8s.io/kubectl => k8s.io/kubectl v0.19.3
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.19.3
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.19.3
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.19.3
	k8s.io/node-api v0.0.0 => k8s.io/node-api v0.19.3
	k8s.io/repo-infra v0.0.0 => k8s.io/repo-infra v0.19.3
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.19.3
	k8s.io/utils v0.0.0 => k8s.io/utils v0.19.3
)
