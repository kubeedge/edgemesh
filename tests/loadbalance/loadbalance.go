package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istio "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var (
	kubeConfigPath      = flag.String("kube-config", "/root/.kube/config", "Use this key to set kube-config path, eg: $HOME/.kube/config")
	namespace           = flag.String("namespace", "default", "All resources namespace")
	deployFile          = flag.String("deployment", "", "Deployment yaml file")
	serviceFile         = flag.String("service", "", "Service yaml file")
	destinationRuleFile = flag.String("destination-rule", "", "Destination rule yaml file")
	counter             = flag.Int("counter", 500, "Test count")
)

type Tester struct {
	sync.Mutex
	Client          *http.Client
	KubeClient      kubernetes.Interface
	IstioClient     istio.Interface
	Deployment      *appsv1.Deployment
	Service         *v1.Service
	DestinationRule *istioapi.DestinationRule
	TestURL         string
	Failure         int
	Counter         int
	AppMap          map[string]int
}

func (t *Tester) Test() {
	var wg sync.WaitGroup
	for i := 0; i < t.Counter; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := t.Client.Get(t.TestURL)
			if err != nil || resp.StatusCode != http.StatusOK {
				t.Failure++
				return
			}
			defer resp.Body.Close()
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Failure++
				return
			}
			app := strings.TrimSpace(string(data))
			t.Lock()
			t.AppMap[app]++
			t.Unlock()
		}()
	}
	wg.Wait()
}

func (t *Tester) PrintResult() {
	fmt.Printf("\nFailure: %d, Success: %d\n", t.Failure, t.Counter-t.Failure)
	for app, count := range t.AppMap {
		fmt.Printf("%s: %d\t(%.1f%%)\n", app, count, float64(count)/float64(t.Counter)*100)
	}
}

func (t *Tester) Cleanup() {
	if t.Deployment != nil {
		err := t.KubeClient.AppsV1().Deployments(t.Deployment.Namespace).Delete(context.Background(), t.Deployment.Name, metav1.DeleteOptions{})
		if err != nil {
			klog.ErrorS(err, "Failed to delete deployment")
		}
	}
	if t.Service != nil {
		err := t.KubeClient.CoreV1().Services(t.Service.Namespace).Delete(context.Background(), t.Service.Name, metav1.DeleteOptions{})
		if err != nil {
			klog.ErrorS(err, "Failed to delete service")
		}
	}
	if t.DestinationRule != nil {
		err := t.IstioClient.NetworkingV1alpha3().DestinationRules(t.DestinationRule.Namespace).Delete(context.Background(), t.DestinationRule.Name, metav1.DeleteOptions{})
		if err != nil {
			klog.ErrorS(err, "Failed to delete destination rule")
		}
	}
	fmt.Println("Cleanup done!")
}

func main() {
	flag.Parse()
	var err error
	t := &Tester{
		Client:  &http.Client{Timeout: 2 * time.Second},
		Counter: *counter,
		AppMap:  make(map[string]int),
	}
	defer t.Cleanup()

	t.KubeClient, t.IstioClient, err = CreateClients(*kubeConfigPath)
	if err != nil {
		klog.ErrorS(err, "Failed to create clients")
		return
	}

	deploy, err := ParseDeployment(*deployFile)
	if err != nil {
		klog.ErrorS(err, "Failed to parse deployment")
		return
	}
	t.Deployment, err = t.KubeClient.AppsV1().Deployments(*namespace).Create(context.Background(), deploy, metav1.CreateOptions{})
	if err != nil {
		klog.ErrorS(err, "Failed to create deployment")
		return
	}

	service, err := ParseService(*serviceFile)
	if err != nil {
		klog.ErrorS(err, "Failed to parse service")
		return
	}
	t.Service, err = t.KubeClient.CoreV1().Services(*namespace).Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		klog.ErrorS(err, "Failed to create service")
		return
	}

	if *destinationRuleFile != "" {
		var dr *istioapi.DestinationRule
		dr, err = ParseDestinationRule(*destinationRuleFile)
		if err != nil {
			klog.ErrorS(err, "Failed to parse destination rule")
			return
		}
		t.DestinationRule, err = t.IstioClient.NetworkingV1alpha3().DestinationRules(*namespace).Create(context.Background(), dr, metav1.CreateOptions{})
		if err != nil {
			klog.ErrorS(err, "Failed to create destination rule")
			return
		}
	}

	// need to wait.
	time.Sleep(5 * time.Second)
	err = WaitForPodBySelectorRunning(t.KubeClient, *namespace, getDeploymentSelector(t.Deployment), 30)
	if err != nil {
		klog.ErrorS(err, "Failed to wait for pods running")
		return
	}
	t.TestURL, err = getTestURL(t.Service)
	if err != nil {
		klog.ErrorS(err, "Failed to get test url")
		return
	}

	t.Test()
	t.PrintResult()
}
