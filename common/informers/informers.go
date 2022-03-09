package informers

import (
	"fmt"
	"sync"

	istio "istio.io/client-go/pkg/clientset/versioned"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	"k8s.io/apimachinery/pkg/runtime"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

type syncedFunc func()

// Manager is an informer factories manager
type Manager struct {
	kubeClient   kubernetes.Interface
	istioClient  istio.Interface
	kubeFactory  k8sinformers.SharedInformerFactory
	istioFactory istioinformers.SharedInformerFactory

	lock        sync.Mutex
	syncedFuncs []syncedFunc
}

func NewManager(config *v1alpha1.KubeAPIConfig) (*Manager, error) {
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

	mgr := Manager{
		kubeClient:   kubeClient,
		istioClient:  istioClient,
		kubeFactory:  k8sinformers.NewSharedInformerFactory(kubeClient, 0),
		istioFactory: istioinformers.NewSharedInformerFactory(istioClient, 0),
	}
	return &mgr, nil
}

// RegisterSyncedFunc add a syncedFunc
func (mgr *Manager) RegisterSyncedFunc(fn syncedFunc) {
	mgr.lock.Lock()
	mgr.syncedFuncs = append(mgr.syncedFuncs, fn)
	mgr.lock.Unlock()
}

// Start starts all factories and run all informers
func (mgr *Manager) Start(stopCh <-chan struct{}) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	mgr.kubeFactory.Start(stopCh)
	mgr.istioFactory.Start(stopCh)

	// sync cache
	for _, ok := range mgr.kubeFactory.WaitForCacheSync(stopCh) {
		if !ok {
			klog.Fatalf("timed out waiting for kubernetes caches to sync")
		}
	}
	for _, ok := range mgr.istioFactory.WaitForCacheSync(stopCh) {
		if !ok {
			klog.Fatalf("timed out waiting for istio caches to sync")
		}
	}

	// when caches are synchronized, all syncedFunc needs to be called
	for _, fn := range mgr.syncedFuncs {
		fn()
	}
}

func (mgr *Manager) GetKubeClient() kubernetes.Interface {
	return mgr.kubeClient
}

func (mgr *Manager) GetIstioClient() istio.Interface {
	return mgr.istioClient
}

func (mgr *Manager) GetKubeFactory() k8sinformers.SharedInformerFactory {
	return mgr.kubeFactory
}

func (mgr *Manager) GetIstioFactory() istioinformers.SharedInformerFactory {
	return mgr.istioFactory
}
