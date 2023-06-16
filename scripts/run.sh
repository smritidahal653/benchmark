#!/bin/bash

echo Applying RBAC
kubectl apply -f ../templates/rbac/
echo

echo Applying high cpu usage workloads
kubectl apply -f ../templates/high_cpu_deployment.yaml
kubectl apply -f ../templates/high_cpu_deployment2.yaml
kubectl top node
echo

echo Applying pod management workload
kubectl apply -f ../templates/deployment.yaml
echo

echo Getting Pod Name
kubectl get pods | grep pod-executor-deployment