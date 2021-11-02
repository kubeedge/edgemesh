package k8s

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeedge/edgemesh/tests/e2e/errors"
	"github.com/kubeedge/kubeedge/tests/e2e/constants"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

const (
	retryTime    = 5
	interValTime = 5 * time.Second
	waitPodTime  = 15 * time.Second
)

var (
	defaultNamespace = "default"
)

// busybox
func generateBusybox(name string, labels, nodeSelector map[string]string) *v1.Pod {
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: v1.PodSpec{
			NodeSelector: nodeSelector,
			Containers: []v1.Container{
				{
					Name:            "busybox",
					Image:           "sequenceiq/busybox",
					ImagePullPolicy: "IfNotPresent",
					Args:            []string{"sleep", "12000"},
				},
			},
		},
	}
}

func CreateBusyboxTool(name string, labels, nodeSelector map[string]string, ctx *utils.TestContext) (*v1.Pod, error) {
	busyboxPod := generateBusybox(name, labels, nodeSelector)
	podURL := ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler
	podBytes, err := json.Marshal(busyboxPod)
	if err != nil {
		utils.Fatalf("Marshalling body failed: %v", err)
		return nil, err
	}
	var pod *v1.Pod
	var podList v1.PodList
	for i := 0; i < retryTime; i++ {
		err = handlePostRequest2K8s(podURL, podBytes)
		if err != nil {
			utils.Fatalf("Frame HTTP request to k8s failed: %v", err)
			return nil, err
		}
		// wait pod ready
		// busyboxPodURL := podURL + "/" + name
		time.Sleep(waitPodTime)
		podList, err = GetPodByLabels(labels, ctx)
		if err != nil {
			utils.Errorf("GetPodByLabels failed: %v", err)
			continue
		}
		if len(podList.Items) == 0 {
			continue
		}
		pod = &podList.Items[0]
		err := WaitforPodsRunning(ctx.Cfg.KubeConfigPath, podList, 240*time.Second)
		if err == nil {
			break
		}
		if errors.IsTimeout(err) {
			err := CleanBusyBoxTool(name, ctx)
			if err != nil {
				utils.Errorf("clean up busybox failed %v", err)
				return nil, err
			}
			utils.Infof("CleanUp finish, Start retry create busybox tool, round %d", i)
			time.Sleep(interValTime)
		}
	}

	return pod, nil
}

func CleanBusyBoxTool(name string, ctx *utils.TestContext) error {
	podURL := ctx.Cfg.K8SMasterForKubeEdge + constants.AppHandler
	resp, err := utils.SendHTTPRequest(http.MethodDelete, podURL+"/"+name)
	if err != nil {
		utils.Fatalf("HTTP request is failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code not equal to %d", http.StatusOK)
	}
	return nil
}

// generate hostname service object
func generateApplication(config *ApplicationConfig) (*appsv1.Deployment, *v1.Service) {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GenDeploymentNameFromUID(config.Name),
			Labels:    config.Labels,
			Namespace: defaultNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: func() *int32 { i := int32(config.Replica); return &i }(),
			Selector: &metav1.LabelSelector{
				MatchLabels: config.Labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: config.Labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            config.Name,
							Image:           config.ImageURL,
							ImagePullPolicy: "IfNotPresent",
							Ports: []v1.ContainerPort{
								{
									ContainerPort: config.ContainerPort,
								},
							},
						},
					},
					NodeSelector: config.NodeSelector,
				},
			},
		},
	}
	service := generateService(config.Name, config.Labels, config.ServicePortName,
		config.ServicePort, config.ServiceProtocol, config.ServiceTargetPort)

	return deployment, service
}

func GenServiceNameFromUID(uid string) string {
	return uid + "-svc"
}

func GenDeploymentNameFromUID(uid string) string {
	return uid
}

type ApplicationConfig struct {
	Name              string
	ImageURL          string
	NodeSelector      map[string]string
	Labels            map[string]string
	Replica           int
	ContainerPort     int32
	ServicePortName   string
	ServicePort       int32
	ServiceProtocol   v1.Protocol
	ServiceTargetPort intstr.IntOrString
	Ctx               *utils.TestContext
}

func createApplication(config *ApplicationConfig) error {
	deployment, service := generateApplication(config)

	deployURL := config.Ctx.Cfg.K8SMasterForKubeEdge + constants.DeploymentHandler
	deployBytes, err := json.Marshal(deployment)
	if err != nil {
		utils.Fatalf("Marshalling body failed: %v", err)
		return err
	}
	for i := 0; i < retryTime; i++ {
		err = handlePostRequest2K8s(deployURL, deployBytes)
		if err != nil {
			utils.Fatalf("Frame HTTP request to k8s failed: %v", err)
			return err
		}
		time.Sleep(waitPodTime)
		// wait deployment ready
		podlist, err := GetPodByLabels(config.Labels, config.Ctx)
		if err != nil {
			utils.Fatalf("GetPods failed: %v", err)
			return err
		}
		err = WaitforPodsRunning(config.Ctx.Cfg.KubeConfigPath, podlist, 240*time.Second)
		if err == nil {
			break
		}
		if errors.IsTimeout(err) {
			utils.Errorf("CreateApplication failed for timeout: %v", err)
			utils.Infof("Start clean up application and retry")
			err := CleanupApplication(config.Name, config.Ctx)
			if err != nil {
				utils.Fatalf("Cleanup Application failed, %v", err)
				return err
			}
			utils.Infof("CleanUp finish, Start retry CreateApplication, round %d", i)
			time.Sleep(interValTime)
		} else {
			utils.Fatalf("CreateApplication failed, %v", err)
			return err
		}
	}

	serviceURL := config.Ctx.Cfg.K8SMasterForKubeEdge + ServiceHandler
	serviceBytes, err := json.Marshal(service)
	if err != nil {
		utils.Fatalf("Marshalling body failed: %v", err)
		return err
	}
	err = handlePostRequest2K8s(serviceURL, serviceBytes)
	if err != nil {
		utils.Fatalf("Frame HTTP request to k8s failed: %v", err)
		return err
	}
	return nil
}

func CleanupApplication(name string, ctx *utils.TestContext) error {
	// fetch pod list for waiting delete
	service, err := GetService(GenServiceNameFromUID(name), ctx)
	if err != nil {
		utils.Fatalf("HTTP request is failed: %v", err)
		return err
	}
	labels := service.Spec.Selector
	podlist, err := GetPodByLabels(labels, ctx)
	if err != nil {
		utils.Fatalf("HTTP request is failed: %v", err)
		return err
	}

	// delete deployment
	deploymentURL := ctx.Cfg.K8SMasterForKubeEdge + constants.DeploymentHandler
	resp, err := utils.SendHTTPRequest(http.MethodDelete, deploymentURL+"/"+GenDeploymentNameFromUID(name))
	if err != nil {
		utils.Fatalf("HTTP request is failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code not equal to %d", http.StatusOK)
	}

	// wait pod delete
	utils.CheckPodDeleteState(ctx.Cfg.K8SMasterForKubeEdge+constants.AppHandler, podlist)

	// delete service
	serviceURL := ctx.Cfg.K8SMasterForKubeEdge + ServiceHandler
	resp, err = utils.SendHTTPRequest(http.MethodDelete, serviceURL+"/"+GenServiceNameFromUID(name))
	if err != nil {
		utils.Fatalf("HTTP request is failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code not equal to %d", http.StatusOK)
	}

	return nil
}

func CreateHostnameApplication(config *ApplicationConfig) error {
	return createApplication(config)
}

func CreateTCPReplyEdgemeshApplication(config *ApplicationConfig) error {
	return createApplication(config)
}

func generateService(name string, selector map[string]string, portName string, port int32,
	protocol v1.Protocol, targetPort intstr.IntOrString) *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GenServiceNameFromUID(name),
			Namespace: defaultNamespace,
		},
		Spec: v1.ServiceSpec{
			Selector: selector,
			Ports: []v1.ServicePort{
				{
					Name:       portName,
					Port:       port,
					Protocol:   protocol,
					TargetPort: targetPort,
				},
			},
		},
	}
}
