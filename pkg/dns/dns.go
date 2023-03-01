package dns

import (
	"os"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/coremain"

	// blank imports to make sure the plugin code is pulled in from vendor
	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/cache"
	_ "github.com/coredns/coredns/plugin/debug"
	_ "github.com/coredns/coredns/plugin/dns64"
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/health"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/kubernetes"
	_ "github.com/coredns/coredns/plugin/loadbalance"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/loop"
	_ "github.com/coredns/coredns/plugin/metrics"
	_ "github.com/coredns/coredns/plugin/pprof"
	_ "github.com/coredns/coredns/plugin/reload"
	_ "github.com/coredns/coredns/plugin/rewrite"
	_ "github.com/coredns/coredns/plugin/template"
	_ "github.com/coredns/coredns/plugin/trace"
	_ "github.com/coredns/coredns/plugin/whoami"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
)

func (d *EdgeDNS) Run() {
	if d.Config.CacheDNS.Enable {
		klog.Infof("Runs CoreDNS v%s as a cache dns", coremain.CoreVersion)
	} else {
		klog.Infof("Runs CoreDNS v%s as a local dns", coremain.CoreVersion)
	}
	corefile, err := caddy.LoadCaddyfile("dns")
	if err != nil {
		klog.Exit(err)
	}
	instance, err := caddy.Start(corefile)
	if err != nil {
		klog.Exit(err)
	}

	if d.Config.KubeAPIConfig.DeleteKubeConfig {
		if err = os.Remove(defaults.TempKubeConfigPath); err != nil {
			klog.Errorf("Failed to delete %s", defaults.TempKubeConfigPath)
		}
	}

	instance.Wait()
}
