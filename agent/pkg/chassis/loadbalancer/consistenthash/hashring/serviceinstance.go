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

package hashring

import (
	"fmt"
	"strings"

	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash"

	"github.com/kubeedge/edgemesh/agent/pkg/chassis/config"
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
	InstanceIP string
}

// String gets service instance key
func (si ServiceInstance) String() string {
	// key format: Namespace#Name#InstanceIP
	return fmt.Sprintf("%s#%s#%s", si.Namespace, si.Name, si.InstanceIP)
}

func SplitKey(key string) (namespace, name, instanceIP string, err error) {
	parts := strings.Split(key, "#")
	if len(parts) != 3 {
		err = fmt.Errorf("invalid ServiceInstance key format")
		return
	}
	return parts[0], parts[1], parts[2], nil
}

func NewServiceInstanceHashRing(instances []ServiceInstance) *consistent.Consistent {
	// create a new consistent instance
	cfg := consistent.Config{
		PartitionCount:    config.Chassis.LoadBalancer.ConsistentHash.PartitionCount,
		ReplicationFactor: config.Chassis.LoadBalancer.ConsistentHash.ReplicationFactor,
		Load:              config.Chassis.LoadBalancer.ConsistentHash.Load,
		Hasher:            defaultHasher{},
	}
	hr := consistent.New(nil, cfg)
	// add service instances to the consistent hash ring
	for _, instance := range instances {
		hr.Add(instance)
	}
	return hr
}
