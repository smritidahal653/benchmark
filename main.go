package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal("could not get config")
	}

	// creates the in-cluster config
	// config, err = rest.InClusterConfig()
	// if err != nil {
	// 	panic(err.Error())
	// }
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	os.Setenv("CONCURRENT_PODS", "10")
	createdPods, err := createPods(clientset)
	if err != nil {
		panic(err.Error())
	}

	err = executeCommandInPod(clientset, config, createdPods)
	if err != nil {
		log.Fatal("Ecountered error while executing command in Pod", "Error:", err)
	} else {
		log.Print("Successfully executed commands for all pods")
	}
}

func createPods(clientset *kubernetes.Clientset) ([]*corev1.Pod, error) {
	numPods, err := strconv.Atoi(os.Getenv("CONCURRENT_PODS"))

	if err != nil {
		return nil, err
	}

	createdPods := make([]*corev1.Pod, 0)
	for i := 0; i < numPods; i++ {
		// Define the pod object
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "example-pod" + fmt.Sprint(i),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "example-container" + fmt.Sprint(i),
						Image: "nginx:latest",
					},
				},
			},
		}

		// Create the pod
		createdPod, err := clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}

		createdPods = append(createdPods, createdPod)
	}

	return createdPods, nil
}

func executeCommandInPod(clientset *kubernetes.Clientset, config *rest.Config, createdPods []*corev1.Pod) error {
	for i := range createdPods {
		// Execute the command inside the pod
		execReq := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(createdPods[i].Name).Namespace(createdPods[i].Namespace).SubResource("exec")
		execReq.VersionedParams(&corev1.PodExecOptions{
			Container: "nginx:latest",
			Command:   []string{"ls"},
			Stdin:     true,
			Stdout:    true,
		}, metav1.ParameterCodec)

		// Create a new executor
		executor, err := remotecommand.NewSPDYExecutor(config, "POST", execReq.URL())
		if err != nil {
			return err
		}

		// Create a new streamer for the command output
		output := &bytes.Buffer{}

		// Execute the command and capture the output
		err = executor.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
			Stdin:  os.Stdin,
			Stdout: output,
			Stderr: os.Stderr,
			Tty:    false,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
