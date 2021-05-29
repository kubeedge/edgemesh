package v1alpha1

import (
	"io/ioutil"

	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

func (c *EdgeMeshConfig) Parse(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		klog.Errorf("Failed to read configfile %s: %v", filename, err)
		return err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		klog.Errorf("Failed to unmarshal configfile %s: %v", filename, err)
		return err
	}
	return nil
}
