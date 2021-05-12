module github.com/kubeedge/edgemesh

go 1.14

require (
	github.com/buraksezer/consistent v0.0.0-20191006190839-693edf70fd72
	github.com/cespare/xxhash v1.1.0
	github.com/go-chassis/go-archaius v0.20.0
	github.com/go-chassis/go-chassis v1.7.1
	github.com/go-chassis/paas-lager v1.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/kubeedge/beehive v0.0.0-20201125122335-cd19bca6e436
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/vishvananda/netlink v1.1.0
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	istio.io/api v0.0.0-20210131044048-bfeb10697307
	istio.io/client-go v0.0.0-20210218000043-b598dd019200
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/component-base v0.19.3
	k8s.io/klog/v2 v2.2.0
	k8s.io/kubernetes v1.19.3
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/kubeedge/edgemesh => ./

replace (
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
