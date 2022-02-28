module github.com/kubeedge/edgemesh

go 1.16

require (
	github.com/buraksezer/consistent v0.9.0
	github.com/cespare/xxhash v1.1.0
	github.com/cespare/xxhash/v2 v2.1.1
	github.com/coredns/coredns v1.8.7
	github.com/go-chassis/go-archaius v0.20.0
	github.com/go-chassis/go-chassis v1.7.1
	github.com/golang/protobuf v1.5.2
	github.com/kubeedge/beehive v0.0.0
	github.com/kubeedge/kubeedge v1.6.2
	github.com/libp2p/go-libp2p v0.13.1-0.20210224102305-f981b25d2738
	github.com/libp2p/go-libp2p-circuit v0.4.0
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-libp2p-quic-transport v0.11.1
	github.com/libp2p/go-libp2p-tls v0.1.3
	github.com/libp2p/go-msgio v0.1.0
	github.com/libp2p/go-ws-transport v0.4.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/opencontainers/runc v1.0.2
	github.com/spf13/cobra v1.2.1
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e
	istio.io/api v0.0.0-20220124163811-3adce9124ae7
	istio.io/client-go v1.12.3
	k8s.io/api v0.23.1
	k8s.io/apimachinery v0.23.1
	k8s.io/apiserver v0.23.0
	k8s.io/client-go v0.23.1
	k8s.io/cloud-provider v0.23.0
	k8s.io/component-base v0.23.0
	k8s.io/klog/v2 v2.40.1
	k8s.io/kubernetes v1.23.0
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/coredns/coredns => github.com/Poorunga/coredns v1.8.8-alpha.1
	github.com/kubeedge/beehive v0.0.0 => github.com/kubeedge/beehive v0.0.0-20201125122335-cd19bca6e436
	github.com/kubeedge/viaduct v0.0.0 => github.com/kubeedge/viaduct v0.0.0-20210601015050-d832643a3d35
	github.com/libp2p/go-libp2p-tls => github.com/Poorunga/go-libp2p-tls v0.1.4
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
)
