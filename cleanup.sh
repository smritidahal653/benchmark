#!/bin/bash

NAMESPACE="default"
LABEL_SELECTOR="for=exec"

kubectl delete pods -l $LABEL_SELECTOR -n $NAMESPACE

