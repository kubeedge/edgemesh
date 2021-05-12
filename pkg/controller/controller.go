package controller

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
	"github.com/kubeedge/edgemesh/pkg/common/informers"
	"github.com/kubeedge/edgemesh/pkg/common/modules"
	"github.com/kubeedge/edgemesh/pkg/controller/config"
)

// Controller list-watch k8s resource object
type Controller struct {
	enable bool
	gc     *GlobalController
}

func newController(enable bool) *Controller {
	if !enable {
		return &Controller{enable: false}
	}
	gc, err := NewGlobalController(informers.GetInformersManager().GetK8sInformerFactory(),
		informers.GetInformersManager().GetIstioInformerFactory())
	if err != nil {
		klog.Fatalf("new process controller failed with error: %s", err)
	}
	return &Controller{
		enable: enable,
		gc:     gc,
	}
}

func Register(c *v1alpha1.Controller) {
	config.InitConfigure(c)
	core.Register(newController(c.Enable))
}

// Name of controller
func (c *Controller) Name() string {
	return modules.ControllerModuleName
}

// Group of controller
func (c *Controller) Group() string {
	return modules.ControllerGroupName
}

// Enable indicates whether enable this module
func (c *Controller) Enable() bool {
	return c.enable
}

// Start controller
func (c *Controller) Start() {
	if err := c.gc.Start(); err != nil {
		klog.Fatalf("start process failed with error: %s", err)
	}
}
