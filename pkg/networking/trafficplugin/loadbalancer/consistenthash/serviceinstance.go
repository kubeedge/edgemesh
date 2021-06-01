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

package consistenthash

import (
	"fmt"
	"strings"

	hashring "github.com/buraksezer/consistent"
	"github.com/cespare/xxhash"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/controller"
	"github.com/kubeedge/edgemesh/pkg/networking/trafficplugin/config"
	"github.com/kubeedge/edgemesh/pkg/networking/util"
)

// default hash algorithm
type defaultHasher struct{}

func (h defaultHasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

// ServiceInstance is the implementation of the consistent.Member interface
type ServiceInstance struct {
	Namespace  string
	Name       string
	InstanceIP string // TODO: Now it is host ip, and later it will be pod ip@Poorunga
}

// String gets service instance key
func (si ServiceInstance) String() string {
	// key format: Namespace.Name.InstanceIP
	return fmt.Sprintf("%s#%s#%s", si.Namespace, si.Name, si.InstanceIP)
}

func newHashRing(instances []ServiceInstance) *hashring.Consistent {
	// create a new consistent instance
	cfg := hashring.Config{
		PartitionCount:    config.Config.LoadBalancer.ConsistentHash.PartitionCount,
		ReplicationFactor: config.Config.LoadBalancer.ConsistentHash.ReplicationFactor,
		Load:              config.Config.LoadBalancer.ConsistentHash.Load,
		Hasher:            defaultHasher{},
	}
	hr := hashring.New(nil, cfg)
	// add service instances to the consistent hash ring
	for _, instance := range instances {
		hr.Add(instance)
	}
	return hr
}

// CreateHashRingByService create and store hash ring by Service
func CreateHashRingByService(svc *v1.Service) {
	// get pods
	pods, err := controller.GetPodLister().Pods(svc.Namespace).List(util.GetPodsSelector(svc))
	if err != nil {
		klog.Errorf("failed to get pod list, reason: %v", err)
		return
	}
	// create service instances
	var instances []ServiceInstance
	for _, p := range pods {
		if p.Status.Phase == v1.PodRunning {
			instances = append(instances, ServiceInstance{
				Namespace:  svc.Namespace,
				Name:       svc.Name,
				InstanceIP: p.Status.HostIP,
			})
		}
	}
	// create hash ring
	hr := newHashRing(instances)
	// store hash ring
	key := fmt.Sprintf("%s.%s", svc.Namespace, svc.Name)
	AddOrUpdateHashRing(key, hr)
}

// UpdateHashRingByService update hash ring by Service
func UpdateHashRingByService(ring string, svc *v1.Service) {
	// get hash ring
	hr, ok := GetHashRing(ring)
	if !ok {
		klog.Errorf("cannot find hash ring %s", ring)
		return
	}
	// get pods
	pods, err := controller.GetPodLister().Pods(svc.Namespace).List(util.GetPodsSelector(svc))
	if err != nil || pods == nil {
		klog.Errorf("failed to get pod list, reason: %v", err)
		return
	}
	added, deleted := lookForDifference(hr, pods, ring)
	for _, key := range added {
		klog.Infof("add ServiceInstance %s to hash ring %s", key, ring)
		namespace, name, instanceIP := splitServiceInstanceKey(key)
		hr.Add(ServiceInstance{
			Namespace:  namespace,
			Name:       name,
			InstanceIP: instanceIP,
		})
	}
	for _, key := range deleted {
		klog.Infof("delete ServiceInstance %s from hash ring %s", key, ring)
		hr.Remove(key)
	}
	// refresh cache
	if len(added) != 0 || len(deleted) != 0 {
		AddOrUpdateHashRing(ring, hr)
	}
}

func splitServiceInstanceKey(key string) (namespace, name, instanceIP string) {
	errMsg := "invalid key format"
	parts := strings.Split(key, "#")
	if len(parts) != 3 {
		klog.Errorf(errMsg)
		return
	}
	return parts[0], parts[1], parts[2]
}

// lookForDifference look for the difference between v1.Pods and HashRing.Members
func lookForDifference(hr *hashring.Consistent, pods []*v1.Pod, key string) ([]string, []string) {
	var src, dest []string
	// get source array from hr.Members
	for _, member := range hr.GetMembers() {
		si, ok := member.(ServiceInstance)
		if !ok {
			klog.Errorf("can't convert to ServiceInstance")
			continue
		}
		src = append(src, si.String())
	}
	klog.Infof("src: %+v", src)
	// build destination array from v1.Pods
	namespace, name := util.SplitServiceKey(key)
	for _, p := range pods {
		if p.DeletionTimestamp != nil {
			continue
		}
		if p.Status.Phase == v1.PodRunning {
			key := fmt.Sprintf("%s#%s#%s", namespace, name, p.Status.HostIP)
			dest = append(dest, key)
		}
	}
	klog.Infof("dest: %+v", dest)
	return arrayCompare(src, dest)
}

// arrayCompare finds the difference between two arrays.
func arrayCompare(src []string, dest []string) ([]string, []string) {
	msrc := make(map[string]byte) // source array set
	mall := make(map[string]byte) // union set
	var set []string              // intersection set

	// 1.Create a map for the source array.
	for _, v := range src {
		msrc[v] = 0
		mall[v] = 0
	}
	// 2.Elements that cannot be stored in the destination array are duplicate elements.
	for _, v := range dest {
		mall[v] = 1
		set = append(set, v)
	}
	// 3.union - intersection = all variable elements
	for _, v := range set {
		delete(mall, v)
	}
	// 4.Now, mall is a complement set, then we use mall to traverse the source array.
	// The element that can be found is the deleted element, and the element that cannot be found is the added element.
	var added, deleted []string
	for v := range mall {
		_, exist := msrc[v]
		if exist {
			deleted = append(deleted, v)
		} else {
			added = append(added, v)
		}
	}
	return added, deleted
}
