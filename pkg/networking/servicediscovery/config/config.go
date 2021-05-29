package config

import (
	"net"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/networking/util"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha1.ServiceDiscovery
	ListenIP net.IP
	Proxy    *net.TCPListener
}

func InitConfigure(s *v1alpha1.ServiceDiscovery) {
	once.Do(func() {
		Config = Configure{
			ServiceDiscovery: *s,
		}
		if Config.Enable {
			// get listen ip
			var err error
			Config.ListenIP, err = util.GetInterfaceIP(Config.ListenInterface)
			if err != nil {
				klog.Errorf("[EdgeMesh] get listen ip err: %v", err)
				return
			}
			// get tcp listener
			tmpPort := 0
			listenAddr := &net.TCPAddr{
				IP:   Config.ListenIP,
				Port: Config.ListenPort + tmpPort,
			}
			for {
				ln, err := net.ListenTCP("tcp", listenAddr)
				if err == nil {
					Config.Proxy = ln
					break
				}
				klog.Warningf("[EdgeMesh] listen on address %v err: %v", listenAddr, err)
				tmpPort++
				listenAddr = &net.TCPAddr{
					IP:   Config.ListenIP,
					Port: Config.ListenPort + tmpPort,
				}
			}
		}
	})
}
