# Kubernetes Pod Management Program

This is a concurrent Go program that demonstrates the creation, execution, and deletion of pods in a Kubernetes cluster using the Kubernetes client library.

## Getting Started

- Ensure you have an active AKS cluster
- Pull the latest image
```
docker pull smritidahal/pod-executor
```
- Go into the scripts folder and provide executable permissions to the two scripts in there 
```
chmod +x cleanup.sh
chmod +x run.sh
```
- Run the run.sh script
```
cd scripts
./run.sh
```
- Copy the pod name from the terminal after script has finished running
-Monitor the logs for the deployment
```
kubectl logs -f <pod_name>
```

## Clean up
- Run the cleanup script
```
./cleanup.sh
```