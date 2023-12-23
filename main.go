package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/gen2brain/beeep"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func watchPods(clientset *kubernetes.Clientset, namespace, deployment string, statusChan chan<- bool) {
	previouslyInBackOff := false

	for {
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: "app=" + deployment,
		})
		if err != nil {
			log.Printf("Error fetching pods: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		inBackOff := false
		for _, pod := range pods.Items {
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil && cs.State.Waiting.Reason == "ImagePullBackOff" {
					fmt.Printf("Pod %s is in ImagePullBackOff status\n", pod.Name)
					inBackOff = true
					break
				}
			}
		}

		if previouslyInBackOff && !inBackOff {
			statusChan <- true
			return
		}

		previouslyInBackOff = inBackOff
		time.Sleep(10 * time.Second)
	}
}

func main() {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespace := "your-namespace"
	deployment := "your-deployment"
	statusChan := make(chan bool)

	go watchPods(clientset, namespace, deployment, statusChan)

	if <-statusChan {
		err = beeep.Notify("Kubernetes Update", "All pods are running normally!", "img.png")
		if err != nil {
			log.Fatal(err)
		}
	}
}
