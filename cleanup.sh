#!/bin/bash

NAMESPACE="default"
LABEL_SELECTOR="for=exec"

kubectl delete -f templates/deployment.yaml
kubectl delete pods -l $LABEL_SELECTOR -n $NAMESPACE

