#!/bin/bash

set -euo pipefail

helm install crossplane \
    --namespace crossplane-system \
    --create-namespace crossplane-stable/crossplane \
	--set args='{"--enable-ssa-claims"}' \
    --version 1.15.0

kubectl -n crossplane-system wait --for=condition=available \
	deployment/crossplane \
	deployment/crossplane-rbac-manager

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

kubectl -n cert-manager wait --for=condition=available \
	deployment/cert-manager \
	deployment/cert-manager-cainjector \
	deployment/cert-manager-webhook
