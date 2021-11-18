package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typecorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	k8slisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

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
			secretOperator: ifm.GetKubeClient().CoreV1().Secrets(constants.SecretNamespace),
		}
		ifm.RegisterInformer(APIConn.secretInformer)
	})
	return APIConn
}

func (c *TunnelServerController) SetPeerAddrInfo(serverAddrName, nodeName string, info *peer.AddrInfo) error {
	serverAddr := make(map[string]*peer.AddrInfo)
	serverAddr[nodeName] = info
	newData, err := json.Marshal(serverAddr)
	if err != nil {
		return fmt.Errorf("Marshal node %s peer info err: %v", nodeName, err)
	}

	secret, err := c.secretLister.Secrets(constants.SecretNamespace).Get(constants.SecretName)
	if errors.IsNotFound(err) {
		newSecret := &apicorev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.SecretName,
				Namespace: constants.SecretNamespace,
			},
			Data: map[string][]byte{},
		}
		newSecret.Data[serverAddrName] = newData
		_, err = c.secretOperator.Create(context.Background(), newSecret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("Create secret %s in %s failed: %v", constants.SecretName, constants.SecretNamespace, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("Get secret %s in %s failed: %v", constants.SecretName, constants.SecretNamespace, err)
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	} else if bytes.Equal(secret.Data[serverAddrName], newData) {
		return nil
	}

	oldData := secret.Data[serverAddrName]
	if oldData == nil {
		secret.Data[serverAddrName] = newData
		secret, err = c.secretOperator.Update(context.Background(), secret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("Update secret %v err: %v", secret, err)
		}
		return nil
	}
	var serverAddrs map[string]*peer.AddrInfo
	err = json.Unmarshal(oldData, &serverAddrs)
	if err != nil {
		klog.Errorf("unmarshal serverAddrInfos %v failed, err: %v", oldData, err)
	}
	serverAddrs[nodeName] = info
	newData, err = json.Marshal(serverAddrs)
	if err != nil {
		klog.Errorf("message serverAddrs failed, err: %v", err)
	}

	secret.Data[serverAddrName] = newData
	secret, err = c.secretOperator.Update(context.Background(), secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("Update secret %v err: %v", secret, err)
	}
	return nil
}
