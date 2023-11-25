package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"k8s.io/klog/v2"
)

// TODO: file log
func Del(args *skel.CmdArgs) (err error) {
	defer func() {
		if e := recover(); e != nil {
			msg := fmt.Sprintf("EdgeMesh CNI panicked during DEL: %s", e)
			if err != nil {
				msg = fmt.Sprintf("%s: error=%s", msg, err)
			}
			err = fmt.Errorf(msg)
		}
		if err != nil {
			klog.Error(err)
		}
	}()

	conf, err := LoadNetConf(args.StdinData)
	if nil != err {
		return fmt.Errorf("failed to load CNI network configuration: %v", err)
	}

	delegateNetconfBytes, err := json.Marshal(conf.Delegate)
	if nil != err {
		return fmt.Errorf("error serializing delegate netconf: %v", err)
	}

	return invoke.DelegateDel(context.TODO(), conf.Delegate["type"].(string), delegateNetconfBytes, nil)
}
