package k8s

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"

	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

func handlePostRequest2K8s(url string, body []byte) error {
	var req *http.Request
	var err error

	defer ginkgo.GinkgoRecover()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
	}
	req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	t := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	utils.Infof("%s %s %v in %v", req.Method, req.URL, resp.Status, time.Since(t))
	return nil
}
