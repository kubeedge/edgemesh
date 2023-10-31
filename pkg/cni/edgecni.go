package cni

import beehiveContext "github.com/kubeedge/beehive/pkg/core/context"

const (
	TCP          = "tcp"
	AgentPodName = "edgemesh-agent"
)

func (cni *EdgeCni) Run() {
	// if Tunmodule start
	if cni.Enable() {
		cni.MeshAdapter.Run()
	}

	<-beehiveContext.Done()
}

func (cni *EdgeCni) CleanupAndExit() error {
	cni.MeshAdapter.CloseRoute()
	return nil
}
