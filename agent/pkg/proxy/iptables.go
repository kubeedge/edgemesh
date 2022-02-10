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
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/controller"
	"github.com/kubeedge/edgemesh/agent/pkg/proxy/protocol"
	"github.com/kubeedge/edgemesh/common/util"
)

const (
	meshRootChain utiliptables.Chain = "EDGE-MESH"
	labelKubeDNS  string             = "k8s-app=kube-dns"
	labelNoProxy  string             = "noproxy=edgemesh"
)

// iptablesJumpChain encapsulates the iptables rule information,
// copy from https://github.com/kubernetes/kubernetes/blob/release-1.18/pkg/proxy/iptables/proxier.go#L368
type iptablesJumpChain struct {
	table     utiliptables.Table
	dstChain  utiliptables.Chain
	srcChain  utiliptables.Chain
	comment   string
	extraArgs []string
}

var rootJumpChains = []iptablesJumpChain{
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
	ignoreRules        []iptablesJumpChain
	expiredIgnoreRules []iptablesJumpChain
	proxyRules         []iptablesJumpChain
	dnatRules          []iptablesJumpChain
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
		ignoreRules:        make([]iptablesJumpChain, 0),
		expiredIgnoreRules: make([]iptablesJumpChain, 0),
		proxyRules:         make([]iptablesJumpChain, 2),
		dnatRules:          make([]iptablesJumpChain, 2),
	}

	// iptables rule cleaning and writing
	proxier.CleanResidue()
	proxier.FlushRules()
	proxier.EnsureRules()

	// set iptables-auto-flush event handler funcs
	controller.APIConn.SetServiceEventHandlers("iptables-auto-flush", cache.ResourceEventHandlerFuncs{
		AddFunc: proxier.svcAdd, UpdateFunc: proxier.svcUpdate, DeleteFunc: proxier.svcDelete})

	return proxier, nil
}

// Start iptables proxy
func (proxier *Proxier) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
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

func (proxier *Proxier) ignoreRuleByService(svc *corev1.Service) iptablesJumpChain {
	var (
		ruleExtraArgs = func(svc *corev1.Service) []string {
			// Headless Service
			switch svc.Spec.ClusterIP {
			case corev1.ClusterIPNone:
				endpoints, err := proxier.kubeClient.CoreV1().Endpoints(svc.Namespace).Get(context.Background(), svc.Name, metav1.GetOptions{})
				if err != nil {
					klog.Errorf("get the endpoints %s failed: %v", strings.Join([]string{svc.Namespace, svc.Name}, "."), err)
					return nil
				}
				endpointIPs := []string{corev1.ClusterIPNone}
				for _, endpointSubset := range endpoints.Subsets {
					epSubset := endpointSubset
					for _, endpointAddress := range epSubset.Addresses {
						epAddress := endpointAddress
						klog.V(4).Infof("eps: %s.%s, endpointAddress.IP:%s", endpoints.Namespace, endpoints.Name, epAddress.IP)
						endpointIPs = append(endpointIPs, epAddress.IP)
					}

					for _, endpointAddress := range epSubset.NotReadyAddresses {
						pod, err := controller.APIConn.GetPodLister().Pods(endpointAddress.TargetRef.Namespace).Get(endpointAddress.TargetRef.Name)
						if err != nil {
							klog.Warningf("get pod %s err: %w", endpointAddress.IP, err)
							continue
						}
						if pod == nil || pod.Status.Phase != corev1.PodRunning {
							klog.V(4).Infof("pod %s is nil or not running.", endpointAddress.IP)
							continue
						}
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
			return fmt.Sprintf("ignore %s/%s service", svc.Namespace, svc.Name)
		}
	)
	return iptablesJumpChain{
		table:     utiliptables.TableNAT,
		dstChain:  "RETURN", // return parent chain
		srcChain:  meshRootChain,
		comment:   ruleComment(svc),
		extraArgs: ruleExtraArgs(svc),
	}
}

// createIgnoreRules exclude some services that must be ignored.
// The following services are ignored by default:
//   - kubernetes.default
//   - coredns.kube-system
//   - kube-dns.kube-system
// In addition, services labeled with noproxy=edgemesh will also be ignored.
func (proxier *Proxier) createIgnoreRules() (ignoreRules, expiredIgnoreRules []iptablesJumpChain, err error) {
	ignoreRulesIptablesEnsureMap := make(map[string]*corev1.Service)

	// kubernetes(kube-apiserver) service
	kubeAPI, err := proxier.kubeClient.CoreV1().Services("default").Get(context.Background(), "kubernetes", metav1.GetOptions{})
	if err != nil && !util.IsNotFoundError(err) {
		klog.Warningf("failed to get kubernetes.default service, err: %v", err)
	} else if err == nil {
		ignoreRulesIptablesEnsureMap[strings.Join([]string{kubeAPI.Namespace, kubeAPI.Name}, ".")] = kubeAPI
	}

	// coredns service
	coreDNS, err := proxier.kubeClient.CoreV1().Services("kube-system").Get(context.Background(), "coredns", metav1.GetOptions{})
	if err != nil && !util.IsNotFoundError(err) {
		klog.Warningf("failed to get coredns.kube-system service, err: %v", err)
	} else if err == nil {
		ignoreRulesIptablesEnsureMap[strings.Join([]string{coreDNS.Namespace, coreDNS.Name}, ".")] = coreDNS
	}

	// kube-dns service
	kubeDNSList, err := proxier.kubeClient.CoreV1().Services("kube-system").List(context.Background(), metav1.ListOptions{LabelSelector: labelKubeDNS})
	if err != nil && !util.IsNotFoundError(err) {
		klog.Warningf("failed to list service labeled with k8s-app=kube-dns, err: %v", err)
	} else if err == nil {
		for _, item := range kubeDNSList.Items {
			klog.V(4).Infof("ignored containing k8s-app=kube-dns label service: %s", item.Name)
			ignoreRulesIptablesEnsureMap[strings.Join([]string{item.Namespace, item.Name}, ".")] = item.DeepCopy()
		}
	}

	// Other services we want to ignore(which service has noproxy=edgemesh label)...
	otherIgnoreServiceList, err := proxier.kubeClient.CoreV1().Services("").List(context.Background(), metav1.ListOptions{LabelSelector: labelNoProxy})
	if err != nil && !util.IsNotFoundError(err) {
		klog.Warningf("failed to list service labeled with noproxy=edgemesh, err: %v", err)
	} else if err == nil {
		for _, item := range otherIgnoreServiceList.Items {
			klog.V(4).Infof("ignored containing noproxy=edgemesh label service: %s", item.Name)
			ignoreRulesIptablesEnsureMap[strings.Join([]string{item.Namespace, item.Name}, ".")] = item.DeepCopy()
		}
	}

	for _, service := range ignoreRulesIptablesEnsureMap {
		ignoreRules = append(ignoreRules, proxier.ignoreRuleByService(service))
	}

	// we need to find out the ignore rules that has expired
	for _, haveIgnoredRule := range proxier.ignoreRules {
		if !funk.Contains(ignoreRules, haveIgnoredRule) {
			expiredIgnoreRules = append(expiredIgnoreRules, haveIgnoredRule)
		}
	}

	return
}

// createProxyRules get proxy rules and DNAT rules
func (proxier *Proxier) createProxyRules() (proxyRules, dnatRules []iptablesJumpChain) {
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
		proxyRules = append(proxyRules, iptablesJumpChain{
			table:     utiliptables.TableNAT,
			dstChain:  newChainName,
			srcChain:  meshRootChain,
			comment:   proxyRuleComment(proto.GetName()),
			extraArgs: proxyRuleArgs(proto.GetName()),
		})
		dnatRules = append(dnatRules, iptablesJumpChain{
			table:     utiliptables.TableNAT,
			dstChain:  "DNAT",
			srcChain:  newChainName,
			extraArgs: dnatRuleArgs(proto.GetName(), proto.GetProxyAddr()),
		})
	}

	return
}

// EnsureRules ensures iptables rules exist
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
	proxier.ignoreRules, proxier.expiredIgnoreRules, err = proxier.createIgnoreRules()
	if err != nil {
		klog.Errorf("failed to create ignore rules: %v", err)
	} else {
		klog.V(5).Infof("ignore rules: %v", proxier.ignoreRules)
	}

	// clean expired ignore rules
	err = proxier.setIgnoreRules("Delete", proxier.expiredIgnoreRules)
	if err != nil {
		klog.Errorf("clean the invalid ignore rules failed: %s", err)
		return
	} else {
		klog.V(5).Infof("clean %d invalid ignore rules.", len(proxier.expiredIgnoreRules))
	}

	// ensure new ignore rules
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
func (proxier *Proxier) setIgnoreRules(ruleSetType string, ignoreRules []iptablesJumpChain) (err error) {
	for _, ignoreRule := range ignoreRules {
		if ignoreRule.extraArgs[0] == corev1.ClusterIPNone {
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
		klog.V(4).ErrorS(err, "Failed flush root chain %s", meshRootChain)
	}

	// flush proxy chains
	for _, rule := range proxier.proxyRules {
		if err := proxier.iptables.FlushChain(rule.table, rule.dstChain); err != nil {
			klog.V(4).ErrorS(err, "Failed flush proxy chain %s", rule.dstChain)
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
		klog.V(4).ErrorS(err, "Failed clean residual non-interface rule %v", nonIfiRuleArgs)
	}
	if err := proxier.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainOutput, nonIfiRuleArgs...); err != nil {
		klog.V(4).ErrorS(err, "Failed clean residual non-interface rule %v", nonIfiRuleArgs)
	}

	// clean up interface iptables rules
	ifiList := []string{"docker0", "cni0"}
	for _, ifi := range ifiList {
		inboundRuleArgs := strings.Split(fmt.Sprintf("-p tcp -d %s -i %s -j EDGE-MESH", proxier.serviceCIDR, ifi), " ")
		if err := proxier.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainPrerouting, inboundRuleArgs...); err != nil {
			klog.V(4).ErrorS(err, "Failed clean residual inbound rule %v", inboundRuleArgs)
		}
		outboundRuleArgs := strings.Split(fmt.Sprintf("-p tcp -d %s -o %s -j EDGE-MESH", proxier.serviceCIDR, ifi), " ")
		if err := proxier.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainOutput, outboundRuleArgs...); err != nil {
			klog.V(4).ErrorS(err, "Failed clean residual outbound rule %v", outboundRuleArgs)
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
				klog.V(4).ErrorS(err, "Failed delete route %v", route)
			}
		}
	}
}

func (proxier *Proxier) svcAdd(obj interface{})               { proxier.EnsureRules() }
func (proxier *Proxier) svcUpdate(oldObj, newObj interface{}) { proxier.EnsureRules() }
func (proxier *Proxier) svcDelete(obj interface{})            { proxier.EnsureRules() }
