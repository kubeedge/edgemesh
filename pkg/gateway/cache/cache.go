package cache

import (
	"sync"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

var (
	secretCache         sync.Map
	virtualServiceCache sync.Map
)

func KeyFormat(namespace, name string) string {
	return namespace + "." + name
}

func UpdateSecret(key string, s *v1.Secret) {
	klog.Infof("Add or update secret %s in cache", key)
	secretCache.Store(key, s)
}

func DeleteSecret(key string) {
	klog.Infof("Delete secret %s from cache", key)
	secretCache.Delete(key)
}

func GetSecret(key string) (*v1.Secret, bool) {
	obj, ok := secretCache.Load(key)
	if !ok {
		klog.Errorf("Secret %s not found", key)
		return nil, false
	}
	s, ok := obj.(*v1.Secret)
	if !ok {
		klog.Errorf("Secret %s type invalid", key)
		return nil, false
	}
	return s, true
}

func RangeSecrets(fn func(key, value interface{}) bool) {
	secretCache.Range(fn)
}

func UpdateVirtualService(key string, vs *v1alpha3.VirtualService) {
	klog.Infof("Add or update virtual service %s in cache", key)
	virtualServiceCache.Store(key, vs)
}

func DeleteVirtualService(key string) {
	klog.Infof("Delete virtual service %s from cache", key)
	virtualServiceCache.Delete(key)
}

func GetVirtualService(key string) (*v1alpha3.VirtualService, bool) {
	obj, ok := virtualServiceCache.Load(key)
	if !ok {
		klog.Errorf("Virtual service %s not found", key)
		return nil, false
	}
	vs, ok := obj.(*v1alpha3.VirtualService)
	if !ok {
		klog.Errorf("Virtual service %s type invalid", key)
		return nil, false
	}
	return vs, true
}

func RangeVirtualServices(fn func(key, value interface{}) bool) {
	virtualServiceCache.Range(fn)
}
