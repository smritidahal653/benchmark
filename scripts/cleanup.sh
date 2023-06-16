#!/bin/bash

NAMESPACE="default"
LABEL_SELECTOR="for=exec"

echo deleting high cpu workloads
kubectl delete -f ../templates/high_cpu_deployment.yaml
kubectl delete -f ../templates/high_cpu_deployment2.yaml
echo

echo deleting pod management workload
kubectl delete -f ../templates/deployment.yaml
echo

echo cleaning up left over pods
kubectl delete pods -l $LABEL_SELECTOR -n $NAMESPACE

