package k8s

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	v1 "k8s.io/api/core/v1"

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
