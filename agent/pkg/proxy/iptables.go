package proxy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thoas/go-funk"
	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/protocol"
	"github.com/kubeedge/edgemesh/common/util"
)

const (
	meshRootChain        utiliptables.Chain = "EDGE-MESH"
	None                                    = "None"
	labelCoreDNS                            = "k8s-app=kube-dns"
	labelNoProxyEdgeMesh                    = "noproxy=edgemesh"
)

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

	invalidIgnoreRules []iptablesEnsureInfo
	invalidProxyRules  []iptablesEnsureInfo
	invalidDNATRules   []iptablesEnsureInfo
}

func NewProxier(subnet string, protoProxies []protocol.ProtoProxy, kubeClient kubernetes.Interface) (proxier *Proxier, err error) {
	primaryProtocol := utiliptables.ProtocolIPv4
	execer := utilexec.New()
	iptInterface := utiliptables.New(execer, primaryProtocol)
	proxier = &Proxier{
		iptables:           iptInterface,
		kubeClient:         kubeClient,
		serviceCIDR:        subnet,
		protoProxies:       protoProxies,
		ignoreRules:        make([]iptablesEnsureInfo, 0),
		proxyRules:         make([]iptablesEnsureInfo, 2),
		dnatRules:          make([]iptablesEnsureInfo, 2),
		invalidIgnoreRules: make([]iptablesEnsureInfo, 0),
	}

	// iptables rule cleaning and writing
	proxier.CleanResidue()
	proxier.FlushRules()
	proxier.EnsureRules()

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
				proxier.FlushRules()
				return
			}
		}
	}()
}

func (proxier *Proxier) ignoreRuleByService(svc *corev1.Service) iptablesEnsureInfo {
	var (
		ruleExtraArgs = func(svc *corev1.Service) []string {
			// Headless Service
			switch svc.Spec.ClusterIP {
			case None:
				endpoints, err := proxier.kubeClient.CoreV1().Endpoints(svc.Namespace).Get(context.Background(), svc.Name, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("get the endpoints %s failed: %v", strings.Join([]string{svc.Namespace, svc.Name}, "."), err)
					return nil
				}
				endpointIPs := []string{None}
				for _, endpointSubset := range endpoints.Subsets {
					epSubset := endpointSubset
					for _, endpointAddress := range epSubset.Addresses {
						epAddress := endpointAddress
						klog.V(4).Infof("eps: %s.%s, endpointAddress.IP:%s", endpoints.Namespace, endpoints.Name, epAddress.IP)
						endpointIPs = append(endpointIPs, epAddress.IP)
					}
				}
				return endpointIPs
			default:
				// cluster ip
				dst := fmt.Sprintf("%s/32", svc.Spec.ClusterIP)
				return []string{"-d", dst}
			}
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
func (proxier *Proxier) createIgnoreRules() (ignoreRules, invalidIgnoreRules []iptablesEnsureInfo, err error) {
	ignoreRulesIptablesEnsureMap := make(map[string]*corev1.Service)
	// kube-apiserver service
	kubeAPI, err := proxier.kubeClient.CoreV1().Services("default").Get(context.Background(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		return
	}
	ignoreRulesIptablesEnsureMap[strings.Join([]string{kubeAPI.Namespace, kubeAPI.Name}, ".")] = kubeAPI

	// kube-dns service
	kubeDNS, err := proxier.kubeClient.CoreV1().Services("kube-system").Get(context.Background(), "kube-dns", metav1.GetOptions{})
	if err != nil {
		klog.Warningf("get kube-dns service failed, your cluster may be not have kube-dns service: %s", err)
	}
	if kubeDNS != nil && err == nil {
		klog.V(4).Infof("ignored kubeDNS: %s", kubeDNS.Name)
		ignoreRulesIptablesEnsureMap[strings.Join([]string{kubeDNS.Namespace, kubeDNS.Name}, ".")] = kubeDNS
	}

	// coredns service
	kubeDNSList, err := proxier.kubeClient.CoreV1().Services("kube-system").List(context.Background(), metav1.ListOptions{LabelSelector: labelCoreDNS})
	if err != nil {
		klog.Warningf("get coredns service failed, your cluster may be not have coredns service: %s", err)
	}
	if err == nil && kubeDNSList != nil && len(kubeDNSList.Items) > 0 {
		for _, item := range kubeDNSList.Items {
			coreDNS := item
			klog.V(4).Infof("ignored containing k8s-app=kube-dns label service: %s", coreDNS.Name)
			ignoreRulesIptablesEnsureMap[strings.Join([]string{item.Namespace, item.Name}, ".")] = &coreDNS
		}
	}

	// Other services we want to ignore...
	otherIgnoreServiceList, err := proxier.kubeClient.CoreV1().Services("").List(context.Background(), metav1.ListOptions{LabelSelector: labelNoProxyEdgeMesh})
	if err != nil {
		klog.Warningf("get Other ignore service failed: %s", err)
	}
	if err == nil && otherIgnoreServiceList != nil && len(otherIgnoreServiceList.Items) > 0 {
		for _, item := range otherIgnoreServiceList.Items {
			otherIgnoreService := item
			klog.V(4).Infof("ignored containing noproxy=edgemesh label service: %s", otherIgnoreService.Name)
			ignoreRulesIptablesEnsureMap[strings.Join([]string{item.Namespace, item.Name}, ".")] = &otherIgnoreService
		}
	}

	for _, service := range ignoreRulesIptablesEnsureMap {
		ignoreRules = append(ignoreRules, proxier.ignoreRuleByService(service))
	}
	// The go-funk library is used here for set operations and comparisons
	for _, haveIgnoredRule := range proxier.ignoreRules {
		if !funk.Contains(ignoreRules, haveIgnoredRule) {
			invalidIgnoreRules = append(invalidIgnoreRules, haveIgnoredRule)
		}
	}

	return
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

	return
}

// ensureRule ensures iptables rules exist
func (proxier *Proxier) EnsureRules() {
	var err error
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

	// recollect need to ignore rules
	proxier.ignoreRules, proxier.invalidIgnoreRules, err = proxier.createIgnoreRules()
	if err != nil {
		klog.Errorf("failed to create ignore rules: %v", err)
	} else {
		klog.V(5).Infof("ignore rules: %v", proxier.ignoreRules)
	}

	// clean the invalid ignore rules
	err = proxier.setIgnoreRules("Delete", proxier.invalidIgnoreRules)
	if err != nil {
		klog.Errorf("clean the invalid ignore rules failed: %s", err)
		return
	} else {
		klog.V(5).Infof("clean %d invalid ignore rules.", len(proxier.invalidIgnoreRules))
	}

	// ensure ignore rules
	err = proxier.setIgnoreRules("Ensure", proxier.ignoreRules)
	if err != nil {
		klog.Errorf("ensure ignore rules failed: %s", err)
		return
	} else {
		klog.V(5).Infof("ensure %d ignore rules.", len(proxier.ignoreRules))
	}

	// recollect need to proxy rules
	proxier.proxyRules, proxier.dnatRules = proxier.createProxyRules()
	klog.V(5).Infof("proxy rules: %v", proxier.proxyRules)
	klog.V(5).Infof("dnat rules: %v", proxier.dnatRules)

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

// setIgnoreRules Delete and Ensure ignore rule for EDGE-MESH chain
func (proxier *Proxier) setIgnoreRules(ruleSetType string, ignoreRules []iptablesEnsureInfo) (err error) {
	for _, ignoreRule := range ignoreRules {
		if ignoreRule.extraArgs[0] == None {
			headLessIps := ignoreRule.extraArgs[1:]
			if len(headLessIps) > 0 {
				for _, headLessIp := range headLessIps {
					hlIp := headLessIp
					args := append([]string{"-d"}, fmt.Sprintf("%s/32", hlIp), "-m", "comment", "--comment", ignoreRule.comment, "-j", string(ignoreRule.dstChain))
					switch ruleSetType {
					case "Ensure":
						if _, err = proxier.iptables.EnsureRule(utiliptables.Prepend, ignoreRule.table, ignoreRule.srcChain, args...); err != nil {
							klog.ErrorS(err, "failed to ensure ignore rules", "table", ignoreRule.table, "srcChain", ignoreRule.srcChain, "dstChain", ignoreRule.dstChain)
						}
					case "Delete":
						if err = proxier.iptables.DeleteRule(ignoreRule.table, ignoreRule.srcChain, args...); err != nil {
							klog.ErrorS(err, "failed to clean invalid ignore rules", "table", ignoreRule.table, "srcChain", ignoreRule.srcChain, "dstChain", ignoreRule.dstChain)
						}
					default:
						return fmt.Errorf("incorrect parameter passing, ruleSetType must be Ensure or Delete")
					}
				}
			}
		} else {
			args := append(ignoreRule.extraArgs,
				"-m", "comment", "--comment", ignoreRule.comment,
				"-j", string(ignoreRule.dstChain),
			)
			switch ruleSetType {
			case "Ensure":
				if _, err = proxier.iptables.EnsureRule(utiliptables.Prepend, ignoreRule.table, ignoreRule.srcChain, args...); err != nil {
					klog.ErrorS(err, "failed to ensure ignore rules", "table", ignoreRule.table, "srcChain", ignoreRule.srcChain, "dstChain", ignoreRule.dstChain)
				}
			case "Delete":
				if err = proxier.iptables.DeleteRule(ignoreRule.table, ignoreRule.srcChain, args...); err != nil {
					klog.ErrorS(err, "failed to clean the invalid ignore rule", "table", ignoreRule.table, "srcChain", ignoreRule.srcChain, "dstChain", ignoreRule.dstChain)
				}
			default:
				return fmt.Errorf("incorrect parameter passing, ruleSetType must be Ensure or Delete")
			}
		}
	}
	return nil
}

// FlushRules flush root chain and proxy chains
func (proxier *Proxier) FlushRules() {
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

// CleanResidue will clean up some iptables or ip routes that may be left on the host
func (proxier *Proxier) CleanResidue() {
	proxier.mu.Lock()
	defer proxier.mu.Unlock()

	// clean up non-interface iptables rules
	nonIfiRuleArgs := strings.Split(fmt.Sprintf("-p tcp -d %s -j EDGE-MESH", proxier.serviceCIDR), " ")
	if err := proxier.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainPrerouting, nonIfiRuleArgs...); err != nil {
		klog.V(4).Error(err, "Failed clean residual non-interface rule %v", nonIfiRuleArgs)
	}
	if err := proxier.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainOutput, nonIfiRuleArgs...); err != nil {
		klog.V(4).Error(err, "Failed clean residual non-interface rule %v", nonIfiRuleArgs)
	}

	// clean up interface iptables rules
	ifiList := []string{"docker0", "cni0"}
	for _, ifi := range ifiList {
		inboundRuleArgs := strings.Split(fmt.Sprintf("-p tcp -d %s -i %s -j EDGE-MESH", proxier.serviceCIDR, ifi), " ")
		if err := proxier.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainPrerouting, inboundRuleArgs...); err != nil {
			klog.V(4).Error(err, "Failed clean residual inbound rule %v", inboundRuleArgs)
		}
		outboundRuleAgrs := strings.Split(fmt.Sprintf("-p tcp -d %s -o %s -j EDGE-MESH", proxier.serviceCIDR, ifi), " ")
		if err := proxier.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainOutput, outboundRuleAgrs...); err != nil {
			klog.V(4).Error(err, "Failed clean residual outbound rule %v", outboundRuleAgrs)
		}
	}

	// clean up ip routes
	dst, err := netlink.ParseIPNet(proxier.serviceCIDR)
	if err != nil {
		klog.Errorf("parse subnet(serviceCIDR) error: %v", err)
		return
	}

	// try to delete the route that may exist
	for _, ifi := range ifiList {
		if gw, err := util.GetInterfaceIP(ifi); err == nil {
			route := netlink.Route{Dst: dst, Gw: gw}
			if err := netlink.RouteDel(&route); err != nil {
				klog.V(4).Error(err, "Failed delete route %v", route)
			}
		}
	}
}
