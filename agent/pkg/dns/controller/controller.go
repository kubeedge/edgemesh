package controller

import (
	"fmt"
	"sync"

	listers "k8s.io/client-go/listers/core/v1"

	"github.com/kubeedge/edgemesh/common/informers"
)

var (
	APIConn *DNSController
	once    sync.Once
)

type DNSController struct {
	svcLister listers.ServiceLister
}

// Init init
func Init(ifm *informers.Manager) {
	once.Do(func() {
		APIConn = &DNSController{
			svcLister: ifm.GetKubeFactory().Core().V1().Services().Lister(),
		}
	})
}

// GetSvcIP get service cluster ip
func (c *DNSController) GetSvcIP(namespace, name string) (ip string, err error) {
	svc, err := c.svcLister.Services(namespace).Get(name)
	if err != nil {
		return "", err
	}
	ip = svc.Spec.ClusterIP
	if ip == "" || ip == "None" {
		return "", fmt.Errorf("service %s.%s no cluster ip", name, namespace)
	}
	return ip, nil
}
