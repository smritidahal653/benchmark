package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	k8sExec "github.com/smritidahal653/benchmark/exec"
	k8sDiscovery "github.com/smritidahal653/benchmark/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ExecutionResult struct {
	PodName      string
	ErrorMessage string
}

func main() {
	// provides the clientset and config
	clientset, config, err := k8sDiscovery.K8s()
	if err != nil {
		log.Fatal(err)
	}

	numPods, err := strconv.Atoi(os.Getenv("NUM_PODS_TO_CREATE"))
	if err != nil {
		log.Print("Could not convert to int. Error: ", err)
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

		// Send stop signal to the goroutines
		close(stopCh)
	}()

	//Timeout after 2 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	//channels to keep track of pod creation, command execution, and deletion
	createResultCh := make(chan bool)
	execResultCh := make(chan ExecutionResult)
	deleteResultCh := make(chan bool)

	//run the workload
	for i := 0; i < numPods; i++ {
		podName := fmt.Sprintf("pod-%d", i)
		pod := createPodObject(podName)
		go runWorkload(clientset, config, pod, createResultCh, execResultCh, deleteResultCh, stopCh)

	}

	// Counters for tracking results
	var podsCreated, commandsExecuted, successfulExecutions, unsuccessfulExecutions, podsDeleted int

	for {
		select {
		case podCreationResult := <-createResultCh:
			if podCreationResult {
				podsCreated++
			}
		case executionResult := <-execResultCh:
			commandsExecuted++
			if executionResult.ErrorMessage != "" {
				unsuccessfulExecutions++
			} else {
				successfulExecutions++
			}
		case podDeletionResult := <-deleteResultCh:
			if podDeletionResult {
				podsDeleted++
			}
		case <-ctx.Done():
			close(stopCh)
			fmt.Println("Finished creating, executing, and deleting pods.")
			fmt.Println("----------------------------------------------------")
			fmt.Printf("Pods created: %d\n", podsCreated)
			fmt.Printf("Commands executed: %d\n", commandsExecuted)
			fmt.Printf("Pods deleted: %d\n", podsDeleted)
			fmt.Println("----------------------------------------------------")
			fmt.Printf("Execution success rate: %d %%\n", ((successfulExecutions / commandsExecuted) * 100))
			return
		case <-stopCh:
			fmt.Println("Stopping pod creation, execution, and deletion...")
			fmt.Println("----------------------------------------------------")
			fmt.Printf("Pods created: %d\n", podsCreated)
			fmt.Printf("Commands executed: %d\n", commandsExecuted)
			fmt.Printf("Pods deleted: %d\n", podsDeleted)
			fmt.Println("----------------------------------------------------")
			fmt.Printf("Execution success rate: %d %%\n", ((successfulExecutions / commandsExecuted) * 100))
			return
		}
	}
}

// returns a pod object with the required service account with the given pod name
func createPodObject(podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   podName,
			Labels: map[string]string{"for": "exec"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "nginx:stable",
				},
			},
			ServiceAccountName: "pod-executor",
		},
	}
}

// creates a pod, execs into the pod then deletes the pod
func runWorkload(clientset *kubernetes.Clientset, config *rest.Config, pod *corev1.Pod, createResultCh chan<- bool, execResultCh chan<- ExecutionResult, deleteResultCh chan<- bool, stopCh <-chan struct{}) {
	//create pod
	createdPod, err := clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create pod %s: %v", pod.Name, err)
	}
	log.Printf("created %s successfully", createdPod.Name)
	createResultCh <- true

	// Wait for the pod to be running
	err = waitForPodRunning(clientset, createdPod.Name)
	if err != nil {
		log.Printf("Pod %s did not start running: %v", createdPod.Name, err)
	}

	// Execute the command inside the pod
	k8s := k8sExec.K8sExec{
		ClientSet:     clientset,
		RestConfig:    config,
		PodName:       createdPod.Name,
		ContainerName: "test-container",
		Namespace:     createdPod.Namespace,
	}

	cmds := []string{"ls"}

	_, stderr, err := k8s.Exec(cmds)

	execResult := ExecutionResult{
		PodName: createdPod.Name,
	}
	if err != nil {
		log.Print("Ecountered error while executing command in Pod ", createdPod.Name, " Error: ", err, string(stderr))
		execResult.ErrorMessage = string(stderr)
	} else {
		log.Print("Successfully executed commands for ", createdPod.Name)
	}
	execResultCh <- execResult

	//Wait for a short duration before deleting the pod
	time.Sleep(1 * time.Second)
	//delete the pod
	err = clientset.CoreV1().Pods("default").Delete(context.TODO(), createdPod.Name, metav1.DeleteOptions{})
	if err != nil {
		log.Print("Encountered error while deleting pod. Error: ", err)
	}
	log.Printf("deleted %s successfully", createdPod.Name)
	deleteResultCh <- true
}

// checks pod status to ensure it is running
func waitForPodRunning(clientset *kubernetes.Clientset, podName string) error {
	for {
		pod, err := clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if pod.Status.Phase == corev1.PodRunning {
			return nil
		}

		time.Sleep(time.Second)
	}
}
