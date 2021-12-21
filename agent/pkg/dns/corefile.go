package dns

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"strings"

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

	"github.com/kubeedge/edgemesh/agent/cmd/edgemesh-agent/app/config"
	"github.com/kubeedge/edgemesh/common/util"
)

// copy from https://github.com/kubernetes/dns/blob/1.21.0/cmd/node-cache/app/configmap.go and update
const (
	stubDomainBlock = `{{.DomainName}}:{{.Port}} {
    bind {{.LocalIP}}
    cache {{.CacheTTL}}
    errors
    forward . {{.UpstreamServers}}
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        {{.KubeAPIServer}}
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
        ttl 30
    }
    log
    loop
    reload
}
`  // cache TTL is 30s by default
	defaultTTL            = 30
	defaultUpstreamServer = "/etc/resolv.conf"
	corefilePath          = "Corefile"
)

// copy from https://github.com/kubernetes/dns/blob/1.21.0/cmd/node-cache/app/configmap.go and update
// stubDomainInfo contains all the parameters needed to compute
// a stubDomain block in the Corefile.
type stubDomainInfo struct {
	DomainName      string
	LocalIP         string
	Port            string
	CacheTTL        int
	UpstreamServers string
	KubeAPIServer   string
}

// copy from https://github.com/kubernetes/dns/blob/1.21.0/cmd/node-cache/app/configmap.go and update
func getStubDomainStr(stubDomainMap map[string][]string, info *stubDomainInfo) (string, error) {
	var tpl bytes.Buffer
	for domainName, servers := range stubDomainMap {
		tmpl, err := template.New("stubDomainBlock").Parse(stubDomainBlock)
		if err != nil {
			return "", fmt.Errorf("failed to create stubDomain template, err : %w", err)
		}
		info.DomainName = domainName
		info.UpstreamServers = strings.Join(servers, " ")
		if err := tmpl.Execute(&tpl, *info); err != nil {
			return "", fmt.Errorf("failed to create stubDomain template, err : %w", err)
		}
	}
	return tpl.String(), nil
}

// copy from https://github.com/kubernetes/dns/blob/1.21.0/cmd/node-cache/app/configmap.go and update
func UpdateCorefile(cfg *config.EdgeMeshAgentConfig) error {
	var apiserver string
	if cfg.CommonConfig.Mode == config.DebugMode {
		if cfg.KubeAPIConfig.Master != "" {
			apiserver = fmt.Sprintf("endpoint %s", cfg.KubeAPIConfig.Master)
		}
		// if kubeconfig is set, use it to overwrite the endpoint
		if cfg.KubeAPIConfig.KubeConfig != "" {
			apiserver = fmt.Sprintf("kubeconfig %s", cfg.KubeAPIConfig.KubeConfig)
		}
	} else if cfg.CommonConfig.Mode == config.EdgeMode {
		apiserver = fmt.Sprintf("endpoint %s", config.DefaultEdgeApiServer)
	}

	// get listen ip
	ListenIP, err := util.GetInterfaceIP(cfg.Modules.EdgeDNSConfig.ListenInterface)
	if err != nil {
		return err
	}

	stubDomainMap := make(map[string][]string)
	stubDomainMap["."] = []string{defaultUpstreamServer}
	stubDomainStr, err := getStubDomainStr(stubDomainMap, &stubDomainInfo{
		LocalIP:       ListenIP.String(),
		Port:          fmt.Sprintf("%d", cfg.Modules.EdgeDNSConfig.ListenPort),
		CacheTTL:      defaultTTL,
		KubeAPIServer: apiserver,
	})
	if err != nil {
		return err
	}

	newConfig := bytes.Buffer{}
	newConfig.WriteString(stubDomainStr)
	if err := ioutil.WriteFile(corefilePath, newConfig.Bytes(), 0666); err != nil {
		return fmt.Errorf("failed to write config file %s - err %w", corefilePath, err)
	}

	return nil
}
