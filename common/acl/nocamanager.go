package acl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/libp2p/go-libp2p-core/crypto"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/certutil"
)

type noCaManager struct {
	keyFile string
}

func init() {
	constructMap[TypeWithNoCA] = newNoCaManager
}

// newNoCaManager creates a noCaManager for edge acl management according to EdgeHub config
func newNoCaManager(tunnel TunnelACLConfig) Manager {
	return &noCaManager{
		keyFile: tunnel.TLSPrivateKeyFile,
	}
}

// Start starts the noCaManager
func (m *noCaManager) Start() {
	// make sure the private key exist
	_, err := os.Stat(m.keyFile)
	if err != nil {
		klog.Infof("Private key does not exist, generate a new one")
		err = m.generateKey()
		if err != nil {
			klog.Fatalf("Error: %v", err)
		}
	} else {
		klog.Infof("Private key exist, skip generate")
	}
}

// generateKey realizes the acl application by token
func (m *noCaManager) generateKey() error {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}
	// save the private key to the file
	if err = certutil.WriteKey(m.keyFile, pk); err != nil {
		return fmt.Errorf("failed to save the private key %s, error: %v", m.keyFile, err)
	}
	return nil
}

func (m *noCaManager) Name() string {
	return "noCaManager"
}

func (m *noCaManager) GetPrivateKey() (crypto.PrivKey, error) {
	certBytes, err := ioutil.ReadFile(m.keyFile)
	if err != nil {
		return nil, fmt.Errorf("read file %s err: %v", m.keyFile, err)
	}

	block, _ := pem.Decode(certBytes)

	privatekey, err := crypto.UnmarshalECDSAPrivateKey(block.Bytes)
	if err != nil {
		return privatekey, fmt.Errorf("unmarshal private key err: %v", err)
	}
	return privatekey, nil
}
