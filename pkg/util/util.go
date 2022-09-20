package util

import (
	"os"

	"github.com/kubeedge/edgemesh/pkg/apis/config/defaults"
)

// DetectRunningMode detects whether the container is running on cloud node or edge node.
// It will recognize whether there is KUBERNETES_PORT in the container environment variable, because
// edged will not inject KUBERNETES_PORT environment variable into the container, but kubelet will.
// What is edged: https://kubeedge.io/en/docs/architecture/edge/edged/
func DetectRunningMode() defaults.RunningMode {
	_, exist := os.LookupEnv("KUBERNETES_PORT")
	if exist {
		return defaults.CloudMode
	}
	return defaults.EdgeMode
}
