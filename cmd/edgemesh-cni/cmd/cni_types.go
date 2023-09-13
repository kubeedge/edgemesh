package cmd

import (
	"encoding/json"
	"net"

	"github.com/containernetworking/cni/pkg/types"
)

const (
	bridge         = "bridge"
	ipamSpiderpool = "spiderpool"
	defaultBrName  = "edgebr0"
)

// K8sArgs is the valid CNI_ARGS used for Kubernetes.
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString //revive:disable-line
	K8S_POD_NAMESPACE          types.UnmarshallableString //revive:disable-line
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString //revive:disable-line
	K8S_POD_UID                types.UnmarshallableString //revive:disable-line
}

type NetConf struct {
	types.NetConf

	Delegate map[string]interface{} `json:"delegate"`
}

func LoadNetConf(argsStdin []byte) (*NetConf, error) {
	n := &NetConf{
		NetConf:  types.NetConf{},
		Delegate: nil,
	}

	err := json.Unmarshal(argsStdin, n)
	if nil != err {
		return n, err
	}

	if n.Delegate == nil {
		n.Delegate = make(map[string]interface{})
	}
	n.Delegate["name"] = n.NetConf.Name
	if !hasItem(n.Delegate, "type") {
		n.Delegate["type"] = bridge
	}
	if !hasItem(n.Delegate, bridge) {
		n.Delegate[bridge] = defaultBrName
	}
	if n.IPAM.Type == "" {
		if !hasItem(n.Delegate, "ipam") {
			n.Delegate["ipam"] = map[string]interface{}{
				"type": ipamSpiderpool,
			}
		}
	} else {
		n.Delegate["ipam"] = map[string]interface{}{
			"type": n.IPAM.Type,
		}
	}

	return n, nil
}

func hasItem(dic map[string]interface{}, key string) bool {
	_, ok := dic[key]
	return ok
}
