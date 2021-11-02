package traffic

import (
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeedge/edgemesh/tests/e2e/k8s"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var (
	defaultNamespace = "default"
)

func CreateHostnameApplication(uid string, nodeSelector map[string]string, servicePort int32, replica int, ctx *utils.TestContext) error {
	imgURL := "k8s.gcr.io/serve_hostname:latest"
	labels := map[string]string{"app": "hostname-edge"}
	config := &k8s.ApplicationConfig{
		Name:              uid,
		ImageURL:          imgURL,
		NodeSelector:      nodeSelector,
		Labels:            labels,
		Replica:           replica,
		ContainerPort:     9376,
		ServicePortName:   "http-0",
		ServicePort:       servicePort,
		ServiceProtocol:   "TCP",
		ServiceTargetPort: intstr.IntOrString{IntVal: 9376},
		Ctx:               ctx,
	}
	return k8s.CreateHostnameApplication(config)
}

func CreateTCPReplyEdgemeshApplication(uid string, nodeSelector map[string]string, servicePort int32, replica int, ctx *utils.TestContext) error {
	imgURL := "kevindavis/tcp-reply-edgemesh:v1.0"
	labels := map[string]string{"app": "tcp-reply-edgemesh-edge"}
	config := &k8s.ApplicationConfig{
		Name:              uid,
		ImageURL:          imgURL,
		NodeSelector:      nodeSelector,
		Labels:            labels,
		Replica:           replica,
		ContainerPort:     9001,
		ServicePortName:   "tcp-0",
		ServicePort:       servicePort,
		ServiceProtocol:   "TCP",
		ServiceTargetPort: intstr.IntOrString{IntVal: 9001},
		Ctx:               ctx,
	}
	return k8s.CreateTCPReplyEdgemeshApplication(config)
}
