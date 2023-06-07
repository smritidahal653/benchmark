package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	k8sExec "github.com/smritidahal653/benchmark/exec"
	k8sDiscovery "github.com/smritidahal653/benchmark/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// provides the clientset and config
	clientset, config, err := k8sDiscovery.K8s()
	if err != nil {
		log.Fatal(err)
	}

	os.Setenv("NUM_PODS_TO_CREATE", "10")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		createPods(clientset)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		executeCommandInPod(clientset, config)
		wg.Done()
	}()

	// wg.Add(1)
	// go func() {
	// 	deletePod(clientset)
	// 	wg.Done()
	// }()

	wg.Wait()
}

// creates number of pods specified by env var NUM_PODS_TO_CREATE
func createPods(clientset *kubernetes.Clientset) {
	numPods, err := strconv.Atoi(os.Getenv("NUM_PODS_TO_CREATE"))
	if err != nil {
		log.Fatal(err)
	}

	for {
		if createdPods := getPodList(clientset, "default"); len(createdPods) < numPods {
			// Define the pod object
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "pod-",
					Labels:       map[string]string{"for": "exec"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "example-container",
							Image: "nginx:latest",
						},
					},
					ServiceAccountName: "pod-executor",
				},
			}

			// Create the pod
			createdPod, err := clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("created %s successfully", createdPod.Name)
		}
		time.Sleep(time.Millisecond * 500)
	}
}

// Execs into a random running pod and runs ls command
func executeCommandInPod(clientset *kubernetes.Clientset, config *rest.Config) {
	for {
		//at least one pod needs to be created first
		if createdPods := getPodList(clientset, "default"); len(createdPods) > 0 {
			podToExecute := randPod(createdPods)

			//can only exec if the pod is ready
			if podToExecute.Status.Phase == corev1.PodRunning {
				// Execute the command inside the pod
				k8s := k8sExec.K8sExec{
					ClientSet:     clientset,
					RestConfig:    config,
					PodName:       podToExecute.Name,
					ContainerName: podToExecute.Spec.Containers[0].Name,
					Namespace:     podToExecute.Namespace,
				}

				cmds := []string{"ls"}

				_, stderr, err := k8s.Exec(cmds)

				if err != nil {
					log.Fatal("Ecountered error while executing command in Pod ", podToExecute.Name, " Error: ", err, string(stderr))
				} else {
					log.Print("Successfully executed commands for ", podToExecute.Name)
				}
			}
		}
		time.Sleep(time.Millisecond * 500)
	}
}

// Deletes a random pod
func deletePod(clientset *kubernetes.Clientset) {
	for {
		//at least one pod needs to be created first
		if createdPods := getPodList(clientset, "default"); len(createdPods) > 0 {
			podToDelete := randPod(createdPods)
			//delete the pod
			err := clientset.CoreV1().Pods("default").Delete(context.TODO(), podToDelete.Name, metav1.DeleteOptions{})
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("deleted %s successfully", podToDelete.Name)
		}
		time.Sleep(time.Millisecond * 500)
	}
}

// Retrieves all pods in the default namespace
func getPodList(clientset *kubernetes.Clientset, namespace string) []corev1.Pod {
	podList := &corev1.PodList{}

	req := clientset.CoreV1().RESTClient().Get().Resource("pods").Namespace(namespace)
	data, err := req.DoRaw(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	if err := json.Unmarshal(data, &podList); err != nil {
		log.Fatal(err)
	}

	return podList.Items
}

func randPod(items []corev1.Pod) *corev1.Pod {
	index := rand.Intn(len(items))

	randPod := items[index]
	return &randPod
}
