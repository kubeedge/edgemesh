package client

import (
	"os"
	"sync"

	istio "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	edgemeshConfig "github.com/kubeedge/edgemesh/pkg/apis/componentconfig/edgemesh/v1alpha1"
)

var emClient *EdgeMeshClient
var once sync.Once

func InitEdgeMeshClient(config *edgemeshConfig.KubeAPIConfig) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags(config.Master, config.KubeConfig)
	if err != nil {
		klog.Errorf("Failed to build config, err: %v", err)
		os.Exit(1)
	}
	kubeConfig.QPS = float32(config.QPS)
	kubeConfig.Burst = int(config.Burst)
	kubeConfig.ContentType = runtime.ContentTypeProtobuf
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	istioKubeConfig := rest.CopyConfig(kubeConfig)
	istioKubeConfig.ContentType = runtime.ContentTypeJSON
	istioClient := istio.NewForConfigOrDie(istioKubeConfig)

	once.Do(func() {
		emClient = &EdgeMeshClient{
			kubeClient:  kubeClient,
			istioClient: istioClient,
		}
	})
}

func GetKubeClient() kubernetes.Interface {
	return emClient.kubeClient
}

func GetIstioClient() istio.Interface {
	return emClient.istioClient
}

type EdgeMeshClient struct {
	kubeClient  *kubernetes.Clientset
	istioClient *istio.Clientset
}
