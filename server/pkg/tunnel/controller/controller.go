package controller

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typecorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8slisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/edgemesh/common/constants"
	"github.com/kubeedge/edgemesh/common/informers"
)

var (
	APIConn *TunnelServerController
	once    sync.Once
)

// TunnelServerController store server addr to secret
type TunnelServerController struct {
	// used to register secret call function
	secretInformer cache.SharedIndexInformer
	// used to get or list secret
	secretLister k8slisters.SecretLister
	// used to add or update or delete secret
	secretOperator typecorev1.SecretInterface
}

func Init(ifm *informers.Manager) *TunnelServerController {
	once.Do(func() {
		kubeFactor := ifm.GetKubeFactory()
		APIConn = &TunnelServerController{
			secretInformer: kubeFactor.Core().V1().Secrets().Informer(),
			secretLister:   kubeFactor.Core().V1().Secrets().Lister(),
			secretOperator: ifm.GetKubeClient().CoreV1().Secrets(constants.SECRET_NAMESPACE),
		}
		ifm.RegisterInformer(APIConn.secretInformer)
	})
	return APIConn
}

func (c *TunnelServerController) SetPeerAddrInfo(nodeName string, info *peer.AddrInfo) error {
	peerAddrINfoBytes, err := info.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Marshal node %s peer info err: %v", nodeName, err)
	}

	secret, err := c.secretLister.Secrets(constants.SECRET_NAMESPACE).Get(constants.SECRET_NAME)
	if errors.IsNotFound(err) {
		newSecret := &apicorev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.SECRET_NAME,
				Namespace: constants.SECRET_NAMESPACE,
			},
			Data: map[string][]byte{},
		}
		newSecret.Data[nodeName] = peerAddrINfoBytes
		newSecret, err = c.secretOperator.Create(context.Background(), newSecret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("Create secret %s in %s failed: %v", constants.SECRET_NAME, constants.SECRET_NAMESPACE, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("Get secret %s in %s failed: %v", constants.SECRET_NAME, constants.SECRET_NAMESPACE, err)
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	} else if bytes.Equal(secret.Data[nodeName], peerAddrINfoBytes) {
		return nil
	}

	secret.Data[nodeName] = peerAddrINfoBytes
	secret, err = c.secretOperator.Update(context.Background(), secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("Update secret %v err: ", secret, err)
	}
	return nil
}
