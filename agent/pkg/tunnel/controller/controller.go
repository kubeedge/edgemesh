package controller

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8slisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/edgemesh/common/constants"
	"github.com/kubeedge/edgemesh/common/informers"
)

var (
	APIConn *TunnelAgentController
	once    sync.Once
)

type TunnelAgentController struct {
	// used to register secret call function
	secretInformer cache.SharedIndexInformer
	// used to get or list secret
	secretLister k8slisters.SecretLister
	// used to add or update or delete secret
	secretOperator corev1.SecretInterface
}

func Init(ifm *informers.Manager) *TunnelAgentController {
	once.Do(func() {
		kubeFactor := ifm.GetKubeFactory()
		APIConn = &TunnelAgentController{
			secretInformer: kubeFactor.Core().V1().Secrets().Informer(),
			secretLister:   kubeFactor.Core().V1().Secrets().Lister(),
			secretOperator: ifm.GetKubeClient().CoreV1().Secrets(constants.SecretNamespace),
		}
	})
	return APIConn
}

func (c *TunnelAgentController) GetPeerAddrInfo(nodeName string) (info *peer.AddrInfo, err error) {
	secret, err := c.secretLister.Secrets(constants.SecretNamespace).Get(constants.SecretName)
	if err != nil {
		return nil, fmt.Errorf("get %s addr from api server err: %w", nodeName, err)
	}

	infoBytes := secret.Data[nodeName]
	if len(infoBytes) == 0 {
		return nil, fmt.Errorf("get %s addr from api server err: %w", nodeName, err)
	}

	info = new(peer.AddrInfo)
	err = info.UnmarshalJSON(infoBytes)
	if err != nil {
		return nil, fmt.Errorf("%s transfer to peer addr info err: %v", string(infoBytes), err)
	}

	return info, nil
}

func (c *TunnelAgentController) SetPeerAddrInfo(nodeName string, info *peer.AddrInfo) error {
	peerAddrINfoBytes, err := info.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal node %s peer info err: %w", nodeName, err)
	}

	secret, err := c.secretLister.Secrets(constants.SecretNamespace).Get(constants.SecretName)
	if err != nil {
		return fmt.Errorf("get secret %s in %s failed: %w", constants.SecretName, constants.SecretNamespace, err)
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	} else if bytes.Equal(secret.Data[nodeName], peerAddrINfoBytes) {
		return nil
	}

	secret.Data[nodeName] = peerAddrINfoBytes
	secret, err = c.secretOperator.Update(context.Background(), secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update secret %v err: %w", secret, err)
	}
	return nil
}
