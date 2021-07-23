package proxy

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
)

const (
	meshChain = "EDGE-MESH"
	rulesFile = "/run/edgemesh-iptables"
)

// iptables rules
type Proxier struct {
	iptables     utiliptables.Interface
	route        netlink.Route
	inboundRule  string
	outboundRule string
	dNatRule     string
}

func newProxier(subnet, netif string, listenIP net.IP, port int) (proxier *Proxier, err error) {
	protocol := utiliptables.ProtocolIPv4
	exec := utilexec.New()
	iptInterface := utiliptables.New(exec, protocol)
	serverAddr := fmt.Sprintf("%s:%d", listenIP.String(), port)
	proxier = &Proxier{
		iptables:     iptInterface,
		inboundRule:  "-p tcp -d " + subnet + " -i " + netif + " -j " + meshChain,
		outboundRule: "-p tcp -d " + subnet + " -o " + netif + " -j " + meshChain,
		dNatRule:     "-p tcp -j DNAT --to-destination " + serverAddr,
	}
	// read and clean iptables rules
	proxier.readAndCleanRule()
	// ensure iptables rules
	proxier.ensureRule()
	// add route
	dst, err := netlink.ParseIPNet(subnet)
	if err != nil {
		return nil, fmt.Errorf("parse subnet error: %v", err)
	}
	proxier.route = netlink.Route{
		Dst: dst,
		Gw:  listenIP,
	}
	err = netlink.RouteAdd(&proxier.route)
	if err != nil {
		klog.Warningf("add route event: %v", err)
	}
	// save iptables rules
	proxier.saveRule()
	return proxier, nil
}

// start network
func (p *Proxier) start() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				p.ensureRule()
			case <-beehiveContext.Done():
				p.clean()
				return
			}
		}
	}()
}

// ensureRule ensures iptables rules exist
func (p *Proxier) ensureRule() {
	iptInterface := p.iptables
	inboundRule := strings.Split(p.inboundRule, " ")
	outboundRule := strings.Split(p.outboundRule, " ")
	dNatRule := strings.Split(p.dNatRule, " ")
	exist, err := iptInterface.EnsureChain(utiliptables.TableNAT, meshChain)
	if err != nil {
		klog.Errorf("ensure chain %s failed with err: %v", meshChain, err)
	}
	if !exist {
		klog.V(4).Infof("chain %s created", meshChain)
	}

	exist, err = iptInterface.EnsureRule(utiliptables.Append, utiliptables.TableNAT, utiliptables.ChainPrerouting, inboundRule...)
	if err != nil {
		klog.Errorf("ensure inbound rule %s failed with err: %v", p.inboundRule, err)
	}
	if !exist {
		klog.V(4).Infof("inbound rule \"%s\" created", p.inboundRule)
	}

	exist, err = iptInterface.EnsureRule(utiliptables.Append, utiliptables.TableNAT, utiliptables.ChainOutput, outboundRule...)
	if err != nil {
		klog.Errorf("ensure outbound rule %s failed with err: %v", p.outboundRule, err)
	}
	if !exist {
		klog.V(4).Infof("outbound rule \"%s\" created", p.outboundRule)
	}

	exist, err = iptInterface.EnsureRule(utiliptables.Append, utiliptables.TableNAT, meshChain, dNatRule...)
	if err != nil {
		klog.Errorf("ensure dnat rule %s failed with err: %v", p.dNatRule, err)
	}
	if !exist {
		klog.V(4).Infof("dnat rule \"%s\" created", p.dNatRule)
	}
}

// saveRule saves iptables rules into file
func (p *Proxier) saveRule() {
	file, err := os.OpenFile(rulesFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		klog.Errorf("open file %s err: %v", rulesFile, err)
		return
	}
	// store
	defer file.Close()
	w := bufio.NewWriter(file)
	fmt.Fprintln(w, p.inboundRule)
	fmt.Fprintln(w, p.dNatRule)
	fmt.Fprintln(w, p.outboundRule)
	w.Flush()
}

// readAndCleanRule reads iptables rules from file and cleans them
func (p *Proxier) readAndCleanRule() {
	exist, err := p.iptables.EnsureChain(utiliptables.TableNAT, meshChain)
	if err != nil {
		klog.Errorf("ensure chain %s failed with err: %v", meshChain, err)
	}
	if !exist {
		if err := p.iptables.FlushChain(utiliptables.TableNAT, meshChain); err != nil {
			klog.Errorf("failed to flush iptables chain, err: %v", err)
		}
		return
	}

	if _, err := os.Stat(rulesFile); err != nil && os.IsNotExist(err) {
		return
	}

	file, err := os.OpenFile(rulesFile, os.O_RDONLY, 0444)
	if err != nil {
		klog.Errorf("open file %s err: %v", rulesFile, err)
		return
	}

	defer file.Close()
	scan := bufio.NewScanner(file)
	scan.Split(bufio.ScanLines)
	for scan.Scan() {
		serverString := scan.Text()
		if strings.Contains(serverString, "-o") {
			if err := p.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainOutput, strings.Split(serverString, " ")...); err != nil {
				klog.Errorf("failed to delete iptables rule, err: %v", err)
				return
			}
		} else if strings.Contains(serverString, "-i") {
			if err := p.iptables.DeleteRule(utiliptables.TableNAT, utiliptables.ChainPrerouting, strings.Split(serverString, " ")...); err != nil {
				klog.Errorf("failed to delete iptables rule, err: %v", err)
				return
			}
		}
	}
	if err := p.iptables.FlushChain(utiliptables.TableNAT, meshChain); err != nil {
		klog.Errorf("failed to flush iptables chain, err: %v", err)
		return
	}
	if err := p.iptables.DeleteChain(utiliptables.TableNAT, meshChain); err != nil {
		klog.Errorf("failed to delete iptables chain, err: %v", err)
		return
	}
}

// clean iptables rule
func (p *Proxier) clean() {
	p.readAndCleanRule()
	if err := netlink.RouteDel(&p.route); err != nil {
		klog.Errorf("delete route err: %v", err)
	}
}
