package k8s

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	v1 "k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var (
	ServiceHandler = "/api/v1/namespaces/default/services"
)

func GetService(name string, ctx *utils.TestContext) (*v1.Service, error) {
	url := ctx.Cfg.K8SMasterForKubeEdge + ServiceHandler + "/" + name
	var service v1.Service
	var resp *http.Response
	var err error
	resp, err = utils.SendHTTPRequest(http.MethodGet, url)
	if err != nil {
		utils.Fatalf("Frame HTTP request failed: %v", err)
		return nil, err
	}

	defer resp.Body.Close()
	contexts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Fatalf("HTTP Response reading has failed: %v", err)
		return nil, err
	}
	err = json.Unmarshal(contexts, &service)
	if err != nil {
		utils.Fatalf("Unmarshal HTTP Response has Failed: %v", err)
		return nil, err
	}
	return &service, nil
}
