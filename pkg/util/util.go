package util

import (
	"io/ioutil"
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
	filePath    = defaults.ConfigDir + "kubeconfig"
	tokenPath   = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// UpdateKubeConfig generate a kubeconfig file and set KubeConfig
func UpdateKubeConfig(c *v1alpha1.KubeAPIConfig) error {
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

	if c.MetaServer.Security.Enable {
		if c.MetaServer.Security.Authorization.RequireAuthorization {
			if !c.MetaServer.Security.Authorization.InsecureSkipTLSVerify {
				// use tls access metaServer
				namedCluster.Cluster.CertificateAuthority = c.MetaServer.Security.Authorization.TLSCaFile
				token, err := getServiceAccountToken()
				if err != nil {
					return err
				}
				namedAuthInfo.AuthInfo.Token = token
				namedAuthInfo.AuthInfo.ClientCertificate = c.MetaServer.Security.Authorization.TLSCertFile
				namedAuthInfo.AuthInfo.ClientKey = c.MetaServer.Security.Authorization.TLSPrivateKeyFile
			} else {
				namedCluster.Cluster.InsecureSkipTLSVerify = true
			}
		} else {
			namedCluster.Cluster.InsecureSkipTLSVerify = true
		}
	}

	kubeConfig := clientcmdv1.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		Clusters:       []clientcmdv1.NamedCluster{namedCluster},
		Contexts:       []clientcmdv1.NamedContext{namedContext},
		CurrentContext: contextName,
		Preferences:    clientcmdv1.Preferences{},
		AuthInfos:      []clientcmdv1.NamedAuthInfo{namedAuthInfo},
	}

	data, err := yaml.Marshal(kubeConfig)
	if err != nil {
		return nil
	}

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0766)
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

	c.KubeConfig = filePath
	return nil
}

func getServiceAccountToken() (string, error) {
	f, err := os.OpenFile(tokenPath, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			klog.ErrorS(err, "close file error")
		}
	}()
	token, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(token), nil
}
