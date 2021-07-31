package acl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"k8s.io/klog/v2"
	"os"

	"github.com/libp2p/go-libp2p-core/crypto"

	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/certutil"
)

type ACLManager struct {
	keyFile string
}

// NewACLManager creates a ACLManager for edge acl management according to EdgeHub config
func NewACLManager(tunnel TunnelACLConfig) *ACLManager {
	return &ACLManager{
		keyFile: tunnel.TLSPrivateKeyFile,
	}
}

// Start starts the ACLManager
func (cm *ACLManager) Start() {
	// make sure the private key exist
	_, err := os.Stat(cm.keyFile)
	if err != nil {
		klog.Infof("Private key does not exist, generate a new one")
		err = cm.generateKey()
		if err != nil {
			klog.Fatalf("Error: %v", err)
		}
	} else {
		klog.Infof("Private key exist, skip generate")
	}
}

// applyCerts realizes the acl application by token
func (cm *ACLManager) generateKey() error {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key, error: %v", err)
	}
	// save the private key to the file
	if err = certutil.WriteKey(cm.keyFile, pk); err != nil {
		return fmt.Errorf("failed to save the private key %s, error: %v", cm.keyFile, err)
	}
	return nil
}

func GetPrivateKey(config TunnelACLConfig) (crypto.PrivKey, error) {
	certManager := NewACLManager(config)
	certManager.Start()

	certBytes, err := ioutil.ReadFile(config.TLSPrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("read file %s err: %v", config.TLSPrivateKeyFile, err)
	}

	block, _ := pem.Decode(certBytes)

	privatekey, err := crypto.UnmarshalECDSAPrivateKey(block.Bytes)
	if err != nil {
		return privatekey, fmt.Errorf("unmarshal private key err: %v", err)
	}
	return privatekey, nil
}
