module github.com/kubeedge/edgemesh

go 1.16

require (
	github.com/buraksezer/consistent v0.9.0
	github.com/cespare/xxhash/v2 v2.1.2
	github.com/coredns/caddy v1.1.0
	github.com/coredns/coredns v1.8.0
	github.com/fsnotify/fsnotify v1.5.4
	github.com/golang/protobuf v1.5.2
	github.com/ipfs/go-datastore v0.5.1
	github.com/ipfs/go-log/v2 v2.5.1
	github.com/kubeedge/beehive v0.0.0
	github.com/kubeedge/kubeedge v1.11.1
	github.com/libp2p/go-libp2p v0.22.0
	github.com/libp2p/go-libp2p-kad-dht v0.18.0
	github.com/libp2p/go-msgio v0.2.0
	github.com/multiformats/go-multiaddr v0.6.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/vishvananda/netlink v1.1.1-0.20201029203352-d40f9887b852
	istio.io/api v0.0.0-20220124163811-3adce9124ae7
	istio.io/client-go v1.12.3
	k8s.io/api v0.23.1
	k8s.io/apimachinery v0.23.1
	k8s.io/client-go v0.23.1
	k8s.io/component-base v0.23.0
	k8s.io/klog/v2 v2.40.1
	k8s.io/kubernetes v1.23.0
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/kubeedge/beehive v0.0.0 => github.com/kubeedge/beehive v0.0.0-20201125122335-cd19bca6e436
	github.com/kubeedge/viaduct v0.0.0 => github.com/kubeedge/viaduct v0.0.0-20210601015050-d832643a3d35
	k8s.io/api v0.0.0 => k8s.io/api v0.23.0
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.23.0
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.23.0
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.23.0
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.23.0
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.23.0
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.23.0
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.23.0
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.23.0
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.23.0
	k8s.io/component-helpers v0.0.0 => k8s.io/component-helpers v0.23.0
	k8s.io/controller-manager v0.0.0 => k8s.io/controller-manager v0.23.0
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.23.0
	k8s.io/csi-api v0.0.0 => k8s.io/csi-api v0.23.0
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.23.0
	k8s.io/gengo v0.0.0 => k8s.io/gengo v0.23.0
	k8s.io/heapster => k8s.io/heapster v1.2.0-beta.1 // indirect
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.30.0
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.23.0
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.23.0
	k8s.io/kube-openapi v0.0.0 => k8s.io/kube-openapi v0.23.0
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.23.0
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.23.0
	k8s.io/kubectl => k8s.io/kubectl v0.23.0
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.23.0
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.23.0
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.23.0
	k8s.io/mount-utils v0.0.0 => k8s.io/mount-utils v0.23.0
	k8s.io/node-api v0.0.0 => k8s.io/node-api v0.23.0
	k8s.io/pod-security-admission v0.0.0 => k8s.io/pod-security-admission v0.23.0
	k8s.io/repo-infra v0.0.0 => k8s.io/repo-infra v0.23.0
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.23.0
	k8s.io/utils v0.0.0 => k8s.io/utils v0.23.0
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client => sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.27
)
