package util

import (
	"os"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

const (
	clusterName = "kubeedge-cluster"
	contextName = "kubeedge-context"
	userName    = "edgemesh"
	saTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

func GenerateKubeClientConfig(c *v1alpha1.KubeAPIConfig) *clientcmdv1.Config {
	namedCluster := clientcmdv1.NamedCluster{
		Name: clusterName,
		Cluster: clientcmdv1.Cluster{
			Server: c.MetaServer.Server,
		},
	}
	namedContext := clientcmdv1.NamedContext{
		Name: contextName,
		Context: clientcmdv1.Context{
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	}
	namedAuthInfo := clientcmdv1.NamedAuthInfo{
		Name:     userName,
		AuthInfo: clientcmdv1.AuthInfo{},
	}

	if c.MetaServer.Security.RequireAuthorization {
		namedAuthInfo.AuthInfo.TokenFile = saTokenPath
		if c.MetaServer.Security.InsecureSkipTLSVerify {
			namedCluster.Cluster.InsecureSkipTLSVerify = true
		} else {
			// use tls access metaServer
			namedCluster.Cluster.CertificateAuthority = c.MetaServer.Security.TLSCaFile
			namedAuthInfo.AuthInfo.ClientCertificate = c.MetaServer.Security.TLSCertFile
			namedAuthInfo.AuthInfo.ClientKey = c.MetaServer.Security.TLSPrivateKeyFile
		}
	}

	return &clientcmdv1.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		Clusters:       []clientcmdv1.NamedCluster{namedCluster},
		Contexts:       []clientcmdv1.NamedContext{namedContext},
		CurrentContext: contextName,
		Preferences:    clientcmdv1.Preferences{},
		AuthInfos:      []clientcmdv1.NamedAuthInfo{namedAuthInfo},
	}
}

func SaveKubeConfigFile(kubeClientConfig *clientcmdv1.Config) error {
	data, err := yaml.Marshal(kubeClientConfig)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(defaults.TempKubeConfigPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			klog.ErrorS(err, "close file error")
		}
	}()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return nil
}
