package acl

import (
	"io/ioutil"

	"github.com/libp2p/go-libp2p-core/crypto"
	"k8s.io/klog/v2"

	meshConstants "github.com/kubeedge/edgemesh/common/constants"
)

type Manager interface {
	Start()
	GetPrivateKey() (crypto.PrivKey, error)
	Name() string
}

type Type int

const (
	TypeWithCA Type = iota
	TypeWithNoCA
)

var constructMap = make(map[Type]func(config TunnelACLConfig) Manager)

func New(tunnel TunnelACLConfig, t Type) Manager {
	construct, ok := constructMap[t]
	if !ok {
		klog.Fatalf("new ACL manager failed because type %d", t)
	}
	m := construct(tunnel)
	klog.Infof("Use %s to handle acl", m.Name())
	return m
}

func NewACLManager(enableSecurity bool, tunnel *TunnelACLConfig) Manager {
	var aclManager Manager
	if enableSecurity {
		// fetch the cloudcore token
		content, err := ioutil.ReadFile(meshConstants.CaServerTokenPath)
		if err != nil {
			klog.Fatalf("failed to read caServerToken from %s, err: %s", meshConstants.CaServerTokenPath, err)
		} else {
			klog.Infof("fetch token from %s success", meshConstants.CaServerTokenPath)
		}
		tunnel.Token = string(content)

		aclManager = New(*tunnel, TypeWithCA)
	} else {
		aclManager = New(*tunnel, TypeWithNoCA)
	}
	return aclManager
}
