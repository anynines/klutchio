#!/bin/bash

# Add Crossplane helm repo and update
helm repo add crossplane-stable https://charts.crossplane.io/stable && helm repo update

# Install Crossplane.
# For --enable-ssa-claims alpha tag see
# server side apply in release v1.17.1
# https://github.com/crossplane/crossplane/releases/tag/v1.17.1
helm install crossplane \
--namespace crossplane-system \
--create-namespace crossplane-stable/crossplane \
--set args='{"--enable-ssa-claims"}' \
--version 1.17.1


# Check if Crossplane developments are ready
kubectl -n crossplane-system wait --for=condition=available deployment/crossplane deployment/crossplane-rbac-manager && echo "Crossplane is successfully deployed"
