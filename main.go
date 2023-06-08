package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	k8sExec "github.com/smritidahal653/benchmark/exec"
	k8sDiscovery "github.com/smritidahal653/benchmark/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	numPods, err := strconv.Atoi(os.Getenv("NUM_PODS_TO_CREATE"))
	if err != nil {
		log.Fatal(err)
	}

	// Create a stop channel to gracefully terminate the program
	stopCh := make(chan struct{})
	defer close(stopCh)

	// Register a signal handler to handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Wait for termination signal
		<-signals
		fmt.Println("Termination signal received. Stopping...")

		// Send stop signal to the goroutines
		close(stopCh)
	}()

	// Run the createPods goroutine
	go createPods(clientset, numPods, stopCh)

	// Wait for a short duration to allow pods to be created
	time.Sleep(5 * time.Second)

	// Run the executeCommands goroutine
	go executeCommandInPod(clientset, config, numPods, stopCh)

	// Wait indefinitely
	select {}
}

// creates number of pods specified by env var NUM_PODS_TO_CREATE
func createPods(clientset *kubernetes.Clientset, numPods int, stopCh <-chan struct{}) {
	var wg sync.WaitGroup
	for i := 1; i <= numPods; i++ {
		wg.Add(1)

		go func(podIndex int) {
			defer wg.Done()

			// Define the pod object
			podName := fmt.Sprintf("pod-%d", podIndex)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   podName,
					Labels: map[string]string{"for": "exec"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-container",
							Image: "nginx:stable",
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
		}(i)

		//Wait for a short duration before creating the next pod
		time.Sleep(1 * time.Second)
	}

	wg.Wait()
}

// Execs into a random running pod and runs ls command
func executeCommandInPod(clientset *kubernetes.Clientset, config *rest.Config, numPods int, stopCh <-chan struct{}) {
	var wg sync.WaitGroup

	for i := 1; i <= numPods; i++ {
		wg.Add(1)

		go func(podIndex int) {
			defer wg.Done()

			podName := fmt.Sprintf("pod-%d", podIndex)
			podToExecute := &corev1.Pod{}
			// Wait until the pod is in the running state
			//err := wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
			podToExecute, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
			if err != nil {
				log.Fatal(err)
			}

			//can only exec if the pod is ready
			if podToExecute.Status.Phase == corev1.PodRunning {
				// Execute the command inside the pod
				k8s := k8sExec.K8sExec{
					ClientSet:     clientset,
					RestConfig:    config,
					PodName:       podToExecute.Name,
					ContainerName: "my-container",
					Namespace:     podToExecute.Namespace,
				}

				cmds := []string{"ls"}

				_, stderr, err := k8s.Exec(cmds)

				if err != nil {
					log.Fatal("Ecountered error while executing command in Pod ", podToExecute.Name, " Error: ", err, string(stderr))
				} else {
					log.Print("Successfully executed commands for ", podToExecute.Name)
				}

				//Wait for a short duration before deleting the pod
				time.Sleep(1 * time.Second)
				//delete the pod
				err = clientset.CoreV1().Pods("default").Delete(context.TODO(), podName, metav1.DeleteOptions{})
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("deleted %s successfully", podName)
			}

		}(i)

		wg.Wait()
	}
}
