# General installation

1. Install prerequisites:
   - a kubernetes cluster
   - crossplane
   - cert-manager

2. Create `local-vars` file from `local-vars.template`

3. Install crossbind components

```bash
kubectl apply -f output/install-v1.0.0-rc1.yaml

kubectl wait --for=condition=Healthy \
	configurations/anynines-dataservices \
	providers/provider-anynines \
	providers/provider-kubernetes
```

4. Install configuration, substituting local settings and secrets

```bash
bash -c "source local-vars; envsubst < output/configure-v1.0.0-rc1.yaml | kubectl apply -f -"

kubectl wait --for=jsonpath='.status.health.lastStatus'=true --timeout 60s \
	providerconfigs/postgresql-service-broker \
	providerconfigs/postgresql-backup-manager
```

---

# Local installation

For a local (development) installation, use `install-dev-*.yaml` instead of `install-*.yaml`.

Next you'll need to create a `local-vars` file, just like on a real setup. The sections below describe how to get the relevant information.

## Kubernetes CA cert and API server address

Extract the information from your kubeconfig:
```
# Set this to *your* cluster's name:
clustername="kind-test-cluster"
yq '.clusters | map(select(.name == "'$clustername'")).0.cluster' < ~/.kube/config
```

Example output:
```
certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUMvakNDQWVhZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRJME1ERXdPREV5TVRjeU9Gb1hEVE0wTURFd05URXlNVGN5T0Zvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTVNJClEzUlh3d0JNS1FNQ1NLQU5nNzA3eFFKYzBXVW1QaW5CRHVhRy9QekVNTTdZczd0dkZvRGNSOTJMcHF3YXVMUGkKMWZzaTZuS0tLTHZCTkpUYWw1Vk16RlFOMmR4QUhWNDNibXJCYU1VSEFoMG4ycG5EZzIvT21MNXlDYlNmYjZRRAp0aHNrRVdRYUs2NmtZbU1vMmRJNy81aGFjSEUwTHdMREJ0aklVcG1xdXBLSldFZHdJV2w1UkJvQllZZi9YbGxyClhYNVpVeHVSYUl6d3FuVXlHRFNqOGZEbzRTR2V6eG5Bb0dlbDNGcHE3RklGTVBRcEp1K3lsTW01ZXBLWk1BQ2cKb0dQNHAzWGlTaE1mMDN3b2R5TjR2dDByNkI0dGU1aXh6OTJVSm1sT3dkbG5mbVc1SCtGUFF6ZUc5Qk02K0p4OApqbFAyQWh6UzdFcEhvTGdFNE8wQ0F3RUFBYU5aTUZjd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0hRWURWUjBPQkJZRUZBZXRNZGc5TmhERnQ5RGVzQ016OExmbkoyNTJNQlVHQTFVZEVRUU8KTUF5Q0NtdDFZbVZ5Ym1WMFpYTXdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBTUJIeUlKQmtvQ2ZrZVdOUWx6LwpsMDFtNHM5bkNud0JSZml5UGp0VkRhaytqTFdYR3VsMUprS3M3MG5nSTBYS1NzNVlsTTQ1citXOHE0eUVmc0d2CjIzQzdKejhOekFtOHhjckJ0NGRzNm1YOTY3RGU0TDhLb1dIdDJFUStjek8wSlhjdjRCc1ZpM1lRdlRCZ0lhZGkKWlJnSUVUZlRRUkJFSjkzTTc4cms2Ri9kUXk2ZHpqbEhPUEIxTmlwTU9OMGZzdXA2bGZUSm1Rb0Q1aEp3OE1XUQp2bnBYdXZYWlhoZGxTMEVnS1hvSmdURGxyUFdxbzZiK3pEQ1YvdjJyK0dUMGVObHZ0aFBKMGJzblFrRWora0MyCkN3SFFQSlpWWXo1Q1hzOTFzZDNFOUV6clVjdGF5OXFZYm5WelROQ2U5d3FYZEhvLzI2YWQ0bks0eWFZeitnVXEKcTVJPQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
server: https://127.0.0.1:65135
```

Copy the values to the appropriate `local-vars`:
```
KUBERNETES_API_EXTERNAL_ADDRESS="<server>"
KUBE_BIND_CA_CERTIFICATE="<certificate-authority-data>"
```

## Cookie encryption & signing keys

Generate two keys, each with:
```
openssl rand -base64 32
```

And set:
```
KUBE_BIND_COOKIE_SIGNING_KEY=<one-key>
KUBE_BIND_COOKIE_ENCRYPTION_KEY=<other-key>
```

## Set up Keycloak

The `install-dev*` bundle also contains keycloak, as a OIDC issuer.

Identify the keycloak pod:
```
pod=$(kubectl get pods -l app=keycloak -oname)
```

Use kcadm.sh within that pod to create a client for the kube-bind backend:
1. Authenticate:
```
kubectl exec -i $pod -- /opt/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080 --realm master --user admin <<EOF
admin
EOF
```
2. Create the client:
```
kubectl exec -i $pod -- /opt/keycloak/bin/kcadm.sh create clients -r master -f - <<EOF
{
  "clientId": "kube-bind-backend",
  "enabled": true,
  "serviceAccountsEnabled": true,
  "authorizationServicesEnabled": true,
  "fullScopeAllowed": true,
  "redirectUris": [
  	"http://dockerhost:9433"
  ]
}
EOF
```
3. Retrieve the client's secret:
```
kubectl exec -i $pod -- /opt/keycloak/bin/kcadm.sh get clients -q clientId=kube-bind-backend | jq '.[0].secret'
```
4. Populate `local-vars` with:
```
KUBE_BIND_OIDC_ISSUER_CLIENT_ID=kube-bind-backend
KUBE_BIND_OIDC_ISSUER_CLIENT_SECRET=<the-secret>
```

Finally forward keycloak port to the host, with
```
kubectl port-forward service/keycloak 8080
```
