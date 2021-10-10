package informers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	istio "istio.io/client-go/pkg/clientset/versioned"
	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	informers   map[string]cache.SharedIndexInformer // key is informer instance address
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
		informers:    make(map[string]cache.SharedIndexInformer),
	}
	return &mgr, nil
}

// RegisterInformer add a informer to Manager. It is important to note that
// the Informer constructed for each resource type will be cached,
// and repeated calls to Informer() on the same resource will return
// the same Informer instance.
func (mgr *Manager) RegisterInformer(informer cache.SharedIndexInformer) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	addr := fmt.Sprintf("%p", informer)
	if _, exist := mgr.informers[addr]; exist {
		return
	}
	mgr.informers[addr] = informer
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

	for addr, informer := range mgr.informers {
		klog.V(4).Infof("informer instance: %s", addr)
		go informer.Run(stopCh)
	}
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

// GetClusterServiceCIDR creates an impossible service to cause an error,
// and obtains cluster-service-ip-range from the error message
func GetClusterServiceCIDR(kubeClient kubernetes.Interface) (string, error) {
	if kubeClient == nil {
		return "", fmt.Errorf("kubeClient is nil")
	}

	badService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bad-service",
		},
		Spec: corev1.ServiceSpec{
			Type:      "ClusterIP",
			ClusterIP: "0.0.0.0", // This is an impossible cluserip
			Ports:     []corev1.ServicePort{{Port: 443}},
		},
	}

	svc, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Create(context.Background(), &badService, metav1.CreateOptions{})
	if err == nil {
		return "", fmt.Errorf("impossible happened, %s was created successfully", svc.Name)
	}

	errMsg := fmt.Sprintf("%v", err)
	errKey := "The range of valid IPs is "
	if ok := strings.Contains(errMsg, errKey); !ok {
		return "", fmt.Errorf("unexpected error: %v", err)
	}

	info := strings.Split(errMsg, errKey)
	if len(info) != 2 {
		return "", fmt.Errorf("invalid info: %v", info)
	}

	return info[1], nil
}
