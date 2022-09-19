package k8s

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/edgemesh/tests/e2e/errors"
	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

func GetPodByLabels(labels map[string]string, ctx *utils.TestContext) (v1.PodList, error) {
	// https://<api-server-ip>/api/v1/namespaces/default/pods?labelSelector=app=busybox,area=east
	url := ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler
	if len(labels) != 0 {
		labelsStr := "?labelSelector="
		for k, v := range labels {
			labelsStr += fmt.Sprintf("%s=%s,", k, v)
		}
		url += labelsStr[:len(labelsStr)-1]
	}

	var pods v1.PodList
	var resp *http.Response
	var err error

	resp, err = utils.SendHTTPRequest(http.MethodGet, url)
	if err != nil {
		utils.Fatalf("Frame HTTP request failed: %v", err)
		return pods, err
	}

	defer resp.Body.Close()
	contexts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Fatalf("HTTP Response reading has failed: %v", err)
		return pods, err
	}
	err = json.Unmarshal(contexts, &pods)
	if err != nil {
		utils.Fatalf("Unmarshal HTTP Response has Failed: %v", err)
		return pods, err
	}
	return pods, nil
}

// WaitforPodsRunning waits util all pods are in running status or timeout
// code copy and add timeout check from https://github.com/kubeedge/kubeedge/blob/master/tests/e2e/utils/pod.go#L196
func WaitforPodsRunning(kubeConfigPath string, podlist v1.PodList, timout time.Duration) error {
	if len(podlist.Items) == 0 {
		return fmt.Errorf("podlist should not be empty")
	}

	podRunningCount := 0
	for _, pod := range podlist.Items {
		if pod.Status.Phase == v1.PodRunning {
			podRunningCount++
		}
	}
	if podRunningCount == len(podlist.Items) {
		utils.Infof("All pods come into running status")
		return nil
	}

	// new kube client
	kubeClient := utils.NewKubeClient(kubeConfigPath)
	// define signal
	signal := make(chan struct{})
	// define list watcher
	listWatcher := cache.NewListWatchFromClient(
		kubeClient.CoreV1().RESTClient(),
		"pods",
		v1.NamespaceAll,
		fields.Everything())
	// new controller
	_, controller := cache.NewInformer(
		listWatcher,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			// receive update events
			UpdateFunc: func(oldObj, newObj interface{}) {
				// check update obj
				p, ok := newObj.(*v1.Pod)
				if !ok {
					utils.Fatalf("Failed to cast observed object to pod")
				}
				// calculate the pods in running status
				count := 0
				for i := range podlist.Items {
					// update pod status in podlist
					if podlist.Items[i].Name == p.Name {
						utils.Infof("PodName: %s PodStatus: %s", p.Name, p.Status.Phase)
						podlist.Items[i].Status = p.Status
					}
					// check if the pod is in running status
					if podlist.Items[i].Status.Phase == v1.PodRunning {
						count++
					}
				}
				// send an end signal when all pods are in running status
				if len(podlist.Items) == count {
					signal <- struct{}{}
				}
			},
		},
	)

	// run controoler
	podChan := make(chan struct{})
	go controller.Run(podChan)
	defer close(podChan)

	// wait for a signal or timeout
	select {
	case _, ok := <-signal:
		if !ok {
			utils.Errorf("chan has been closed")
		}
		utils.Infof("All pods come into running status")
	case <-time.After(timout):
		errInfo := fmt.Sprintf("Wait for pods come into running status timeout: %v", timout)
		return errors.NewTimeoutErr(errInfo)
	}
	return nil
}
