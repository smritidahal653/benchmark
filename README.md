# Kubernetes Pod Management Program

This is a concurrent Go program that demonstrates the creation, execution, and deletion of pods in a Kubernetes cluster using the Kubernetes client library.

## Getting Started

- Ensure you have an active AKS cluster
- Pull the latest image
```
docker pull smritidahal/pod-executor
```
- Apply all files under the templates/rbac folder 
```
kubectl apply -f templates/rbac/
```
- Apply the two high cpu workload
```
kubectl apply -f templates/high_cpu_deployment.yanl
kubectl apply -f templates/high_cpu_deployment2.yaml
```
- Verify cpu usage 
```
kubectl top node
```
- Update the environment variables in deployment.yaml file according to your needs
- Apply the deployment file 
```
kubectl apply -f templates/deployment.yaml
```
-Monitor the logs for the deployment
```
kubectl logs -f <pod_name>
```

## Clean up
- Run the cleanup script
```
./cleanup.sh
```