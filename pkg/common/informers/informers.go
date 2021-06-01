/*
Copyright 2021 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package informers

import (
	"sync"
	"time"

	istioinformers "istio.io/client-go/pkg/informers/externalversions"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/common/client"
)

type Manager interface {
	GetK8sInformerFactory() k8sinformers.SharedInformerFactory
	GetIstioInformerFactory() istioinformers.SharedInformerFactory
	Start(stopCh <-chan struct{})
}

type informers struct {
	defaultResync              time.Duration
	keClient                   kubernetes.Interface
	lock                       sync.Mutex
	informers                  map[string]cache.SharedIndexInformer
	k8sSharedInformerFactory   k8sinformers.SharedInformerFactory
	istioSharedInformerFactory istioinformers.SharedInformerFactory
}

var globalInformers Manager
var once sync.Once

func GetInformersManager() Manager {
	once.Do(func() {
		globalInformers = &informers{
			defaultResync:              0,
			keClient:                   client.GetKubeClient(),
			informers:                  make(map[string]cache.SharedIndexInformer),
			k8sSharedInformerFactory:   k8sinformers.NewSharedInformerFactory(client.GetKubeClient(), 0),
			istioSharedInformerFactory: istioinformers.NewSharedInformerFactory(client.GetIstioClient(), 0),
		}
	})
	return globalInformers
}

func (ifs *informers) GetK8sInformerFactory() k8sinformers.SharedInformerFactory {
	return ifs.k8sSharedInformerFactory
}

func (ifs *informers) GetIstioInformerFactory() istioinformers.SharedInformerFactory {
	return ifs.istioSharedInformerFactory
}

func (ifs *informers) Start(stopCh <-chan struct{}) {
	ifs.lock.Lock()
	defer ifs.lock.Unlock()

	for name, informer := range ifs.informers {
		klog.V(5).Infof("start informer %s", name)
		go informer.Run(stopCh)
	}
	ifs.k8sSharedInformerFactory.Start(stopCh)
	ifs.istioSharedInformerFactory.Start(stopCh)
}
