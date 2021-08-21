package proxy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/protocol"
)

const meshRootChain utiliptables.Chain = "EDGE-MESH"

type iptablesEnsureInfo struct {
	table     utiliptables.Table
	dstChain  utiliptables.Chain
	srcChain  utiliptables.Chain
	comment   string
	extraArgs []string
}

var rootJumpChains = []iptablesEnsureInfo{
	{utiliptables.TableNAT, meshRootChain, utiliptables.ChainOutput, "edgemesh root chain", nil},
	{utiliptables.TableNAT, meshRootChain, utiliptables.ChainPrerouting, "edgemesh root chain", nil},
}

type Proxier struct {
	mu sync.Mutex

	iptables   utiliptables.Interface
	kubeClient kubernetes.Interface

	// serviceCIDR is kubernetes service-cluster-ip-range
	serviceCIDR string

	// protoProxies represents the protocol that requires proxy
	protoProxies []protocol.ProtoProxy

	// iptables rules
	ignoreRules []iptablesEnsureInfo
	proxyRules  []iptablesEnsureInfo
	dnatRules   []iptablesEnsureInfo
}

func NewProxier(subnet string, protoProxies []protocol.ProtoProxy, kubeClient kubernetes.Interface) (proxier *Proxier, err error) {
	primaryProtocol := utiliptables.ProtocolIPv4
	execer := utilexec.New()
	iptInterface := utiliptables.New(execer, primaryProtocol)
	proxier = &Proxier{
		iptables:     iptInterface,
		kubeClient:   kubeClient,
		serviceCIDR:  subnet,
		protoProxies: protoProxies,
		ignoreRules:  make([]iptablesEnsureInfo, 2),
		proxyRules:   make([]iptablesEnsureInfo, 2),
		dnatRules:    make([]iptablesEnsureInfo, 2),
	}

	// Initialize iptables rules
	if err = proxier.InitRules(); err != nil {
		return proxier, err
	}
	proxier.CleanRules()
	proxier.EnsureRules()
	// TODO(Poorunga) delete this code
	klog.V(5).Infof("ntf router: %d", netlink.NTF_ROUTER)
	return proxier, nil
}

// Start iptables proxy
func (proxier *Proxier) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <-ticker.C:
				proxier.EnsureRules()
			case <-beehiveContext.Done():
				proxier.CleanRules()
				return
			}
		}
	}()
}

func (proxier *Proxier) InitRules() (err error) {
	proxier.ignoreRules, err = proxier.createIgnoreRules()
	if err != nil {
		return fmt.Errorf("failed to create ignore rules: %v", err)
	} else {
		klog.V(5).Infof("ignore rules: %v", proxier.ignoreRules)
	}

	proxier.proxyRules, proxier.dnatRules = proxier.createProxyRules()
	klog.V(5).Infof("proxy rules: %v", proxier.proxyRules)
	klog.V(5).Infof("dnat rules: %v", proxier.dnatRules)

	return nil
}

func ignoreRuleByService(svc *corev1.Service) iptablesEnsureInfo {
	var (
		ruleExtraArgs = func(svc *corev1.Service) []string {
			dst := fmt.Sprintf("%s/32", svc.Spec.ClusterIP)
			return []string{"-d", dst}
		}

		ruleComment = func(svc *corev1.Service) string {
			return fmt.Sprintf("ignore %s.%s service", svc.Name, svc.Namespace)
		}
	)
	return iptablesEnsureInfo{
		table:     utiliptables.TableNAT,
		dstChain:  "RETURN", // return parent chain
		srcChain:  meshRootChain,
		comment:   ruleComment(svc),
		extraArgs: ruleExtraArgs(svc),
	}
}

// createIgnoreRules exclude some services that must be ignored
func (proxier *Proxier) createIgnoreRules() (ignoreRules []iptablesEnsureInfo, err error) {
	// kube-apiserver service
	kubeAPI, err := proxier.kubeClient.CoreV1().Services("default").Get(context.Background(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		return
	}
	ignoreRules = append(ignoreRules, ignoreRuleByService(kubeAPI))

	// kube-dns service
	kubeDNS, err := proxier.kubeClient.CoreV1().Services("kube-system").Get(context.Background(), "kube-dns", metav1.GetOptions{})
	if err != nil {
		return
	}
	ignoreRules = append(ignoreRules, ignoreRuleByService(kubeDNS))

	// Other services we want to ignore...

	return ignoreRules, nil
}

// createProxyRules get proxy rules and DNAT rules
func (proxier *Proxier) createProxyRules() (proxyRules, dnatRules []iptablesEnsureInfo) {
	var (
		newChainName utiliptables.Chain

		proxyChainName = func(protoName protocol.ProtoName) utiliptables.Chain {
			return utiliptables.Chain(fmt.Sprintf("%s-%s", meshRootChain, strings.ToUpper(string(protoName))))
		}

		proxyRuleComment = func(protoName protocol.ProtoName) string {
			return fmt.Sprintf("%s service proxy", string(protoName))
		}

		proxyRuleArgs = func(protoName protocol.ProtoName) []string {
			return []string{"-p", string(protoName), "-d", proxier.serviceCIDR}
		}

		dnatRuleArgs = func(protoName protocol.ProtoName, serverAddr string) []string {
			return []string{"-p", string(protoName), "-j", "DNAT", "--to-destination", serverAddr}
		}
	)

	for _, proto := range proxier.protoProxies {
		newChainName = proxyChainName(proto.GetName())
		proxyRules = append(proxyRules, iptablesEnsureInfo{
			table:     utiliptables.TableNAT,
			dstChain:  newChainName,
			srcChain:  meshRootChain,
			comment:   proxyRuleComment(proto.GetName()),
			extraArgs: proxyRuleArgs(proto.GetName()),
		})
		dnatRules = append(dnatRules, iptablesEnsureInfo{
			table:     utiliptables.TableNAT,
			dstChain:  "DNAT",
			srcChain:  newChainName,
			extraArgs: dnatRuleArgs(proto.GetName(), proto.GetProxyAddr()),
		})
	}

	return proxyRules, dnatRules
}

// ensureRule ensures iptables rules exist
func (proxier *Proxier) EnsureRules() {
	proxier.mu.Lock()
	defer proxier.mu.Unlock()

	// ensure root jump chains
	for _, jump := range rootJumpChains {
		if _, err := proxier.iptables.EnsureChain(jump.table, jump.dstChain); err != nil {
			klog.ErrorS(err, "Failed to ensure chain exists", "table", jump.table, "chain", jump.dstChain)
			return
		}
		args := append(jump.extraArgs,
			"-m", "comment", "--comment", jump.comment,
			"-j", string(jump.dstChain),
		)
		if _, err := proxier.iptables.EnsureRule(utiliptables.Prepend, jump.table, jump.srcChain, args...); err != nil {
			klog.ErrorS(err, "Failed to ensure root jump chains", "table", jump.table, "srcChain", jump.srcChain, "dstChain", jump.dstChain)
			return
		}
	}

	// ensure ignore rules
	for _, rule := range proxier.ignoreRules {
		args := append(rule.extraArgs,
			"-m", "comment", "--comment", rule.comment,
			"-j", string(rule.dstChain),
		)
		if _, err := proxier.iptables.EnsureRule(utiliptables.Append, rule.table, rule.srcChain, args...); err != nil {
			klog.ErrorS(err, "Failed to ensure ignore rules", "table", rule.table, "srcChain", rule.srcChain, "dstChain", rule.dstChain)
			return
		}
	}

	// ensure proxy rules
	for _, jump := range proxier.proxyRules {
		if _, err := proxier.iptables.EnsureChain(jump.table, jump.dstChain); err != nil {
			klog.ErrorS(err, "Failed to ensure chain exists", "table", jump.table, "chain", jump.dstChain)
			return
		}
		args := append(jump.extraArgs,
			"-m", "comment", "--comment", jump.comment,
			"-j", string(jump.dstChain),
		)
		if _, err := proxier.iptables.EnsureRule(utiliptables.Append, jump.table, jump.srcChain, args...); err != nil {
			klog.ErrorS(err, "Failed to ensure proxy rules", "table", jump.table, "srcChain", jump.srcChain, "dstChain", jump.dstChain)
			return
		}
	}

	// ensure dnat rules
	for _, rule := range proxier.dnatRules {
		if _, err := proxier.iptables.EnsureRule(utiliptables.Append, rule.table, rule.srcChain, rule.extraArgs...); err != nil {
			klog.ErrorS(err, "Failed to ensure dnat rules", "table", rule.table, "srcChain", rule.srcChain, "dstChain", rule.dstChain)
			return
		}
	}
}

// CleanRules flush root chain and proxy chains
func (proxier *Proxier) CleanRules() {
	proxier.mu.Lock()
	defer proxier.mu.Unlock()

	// flush root chain
	if err := proxier.iptables.FlushChain(utiliptables.TableNAT, meshRootChain); err != nil {
		klog.V(4).Error(err, "Failed flush root chain %s", meshRootChain)
	}

	// flush proxy chains
	for _, rule := range proxier.proxyRules {
		if err := proxier.iptables.FlushChain(rule.table, rule.dstChain); err != nil {
			klog.V(4).Error(err, "Failed flush proxy chain %s", rule.dstChain)
		}
	}
}
