#!/bin/bash

# Add Crossplane helm repo and update
helm repo add crossplane-stable https://charts.crossplane.io/stable && helm repo update

# Install Crossplane v2.
helm install crossplane \
--namespace crossplane-system \
--create-namespace crossplane-stable/crossplane \
--version 2.2.1 \
--wait --timeout 5m

# Check if Crossplane deployments are ready
kubectl -n crossplane-system wait --for=condition=available deployment/crossplane deployment/crossplane-rbac-manager && echo "Crossplane is successfully deployed"
