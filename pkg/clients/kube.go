package clients

import (
	"fmt"

	istio "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

type Clients struct {
	Config      *v1alpha1.KubeAPIConfig
	kubeClient  kubernetes.Interface
	istioClient istio.Interface
}

func NewClients(config *v1alpha1.KubeAPIConfig) (*Clients, error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config, %v", err)
	}
	kubeConfig.QPS = float32(config.QPS)
	kubeConfig.Burst = int(config.Burst)
	kubeConfig.ContentType = config.ContentType
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	istioKubeConfig := rest.CopyConfig(kubeConfig)
	istioKubeConfig.ContentType = runtime.ContentTypeJSON
	istioClient := istio.NewForConfigOrDie(istioKubeConfig)

	return &Clients{
		Config:      config,
		kubeClient:  kubeClient,
		istioClient: istioClient,
	}, nil
}

func (cli *Clients) GetKubeClient() kubernetes.Interface {
	return cli.kubeClient
}

func (cli *Clients) GetIstioClient() istio.Interface {
	return cli.istioClient
}
