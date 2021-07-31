package tunnel

import (
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/agent/pkg/tunnel/config"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/certificate"
	"github.com/kubeedge/edgemesh/pkg/common/modules"
)

type Tunnel struct {
	certManager certificate.CertManager
	enable      bool
}

func NewTunnel(enable bool) *Tunnel {
	return &Tunnel{
		enable:      enable,
	}
}

func Register(tl *v1alpha1.Tunnel) {
	config.InitConfigure(tl)
	core.Register(NewTunnel(tl.Enable))
}

func (t *Tunnel) Name() string {
	return modules.AgentTunnelModuleName
}

func (t *Tunnel) Group() string {
	return modules.AgentTunnelGroupName
}

func (t *Tunnel) Enable() bool {
	return t.enable
}

func (t *Tunnel) Start() {
	certificateConfig := certificate.TunnelCertificate{
		Heartbeat:          config.Config.Heartbeat,
		TLSCAFile:          config.Config.TLSCAFile,
		TLSCertFile:        config.Config.TLSCertFile,
		TLSPrivateKeyFile:  config.Config.TLSPrivateKeyFile,
		Token:              config.Config.Token,
		HTTPServer:         config.Config.HTTPServer,
		RotateCertificates: config.Config.RotateCertificates,
		HostnameOverride:   config.Config.HostnameOverride,
	}
	t.certManager = certificate.NewCertManager(certificateConfig, config.Config.NodeName)
	t.certManager.Start()

	// TODO ifRotationDone() ????, 后面要添加这个东西，如果证书轮换了，要重新进行连接
	select {}
}