package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istio "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

var ErrPodCompleted = errors.New("pod completed")

func CreateClients(kubeconfig string) (kubernetes.Interface, istio.Interface, error) {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, err
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	istioClient := istio.NewForConfigOrDie(kubeConfig)
	return kubeClient, istioClient, nil
}

// ParseDeployment parse the deployment from yaml file.
func ParseDeployment(filepath string) (*appsv1.Deployment, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var deploy appsv1.Deployment
	err = yaml.Unmarshal(data, &deploy)
	if err != nil {
		return nil, err
	}
	return &deploy, nil
}

// ParseService parse the service from yaml file.
func ParseService(filepath string) (*v1.Service, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var service v1.Service
	err = yaml.Unmarshal(data, &service)
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// ParseDestinationRule parse the destination rule from yaml file.
func ParseDestinationRule(filepath string) (*istioapi.DestinationRule, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var dr istioapi.DestinationRule
	err = yaml.Unmarshal(data, &dr)
	if err != nil {
		return nil, err
	}
	return &dr, nil
}

func WaitForPodBySelectorRunning(c kubernetes.Interface, namespace, selector string, timeout int) error {
	podList, err := ListPods(c, namespace, selector)
	if err != nil {
		return err
	}
	if len(podList.Items) == 0 {
		return fmt.Errorf("no pods in %s with selector %s", namespace, selector)
	}

	for _, pod := range podList.Items {
		if err := waitForPodRunning(c, namespace, pod.Name, time.Duration(timeout)*time.Second); err != nil {
			return err
		}
	}
	return nil
}

func ListPods(c kubernetes.Interface, namespace, selector string) (*v1.PodList, error) {
	listOptions := metav1.ListOptions{LabelSelector: selector}
	podList, err := c.CoreV1().Pods(namespace).List(context.Background(), listOptions)
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func waitForPodRunning(c kubernetes.Interface, namespace, podName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, isPodRunning(c, podName, namespace))
}

func isPodRunning(c kubernetes.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, ErrPodCompleted
		}
		return false, nil
	}
}

func getDeploymentSelector(deploy *appsv1.Deployment) string {
	selector := []string{}
	for key, val := range deploy.Spec.Selector.MatchLabels {
		selector = append(selector, fmt.Sprintf("%s=%s", key, val))
	}
	return strings.Join(selector, ",")
}

func getTestURL(service *v1.Service) (string, error) {
	clusterIP := service.Spec.ClusterIP
	if clusterIP == "" || clusterIP == v1.ClusterIPNone {
		return "", fmt.Errorf("no Cluster IP")
	}
	if len(service.Spec.Ports) == 0 {
		return "", fmt.Errorf("not Ports")
	}
	return fmt.Sprintf("http://%s:%d", clusterIP, service.Spec.Ports[0].Port), nil
}
