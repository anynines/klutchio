# Konnector Control Plane Mode

## Overview

By default, the konnector (the controller that syncs Kubernetes resources between the app cluster and the control plane) runs on the app cluster. This facilitates trust for the user because they can verify that the exact code that is available open source is running in their cluster.

However, it can also be advantageous to run the konnector on the control plane instead. Benefits include:

- **Reduced footprint**: Minimizes the number of components running in the app cluster
- **Enhanced security**: Hides the privileged service account token that the konnector uses to authenticate to the control plane
- **Centralized management**: Platform operators have more control over bindings and connectivity

## Architecture

### Cluster Roles by Mode

The **binding cluster** is a logical role and depends on the mode:

- **Default mode**: binding cluster = app cluster
- **Control plane mode**: binding cluster = control plane cluster

`APIServiceBinding` objects and the `APIServiceBinding` CRD always belong to the **binding cluster**.

### Default Mode (Konnector on App Cluster)

```
┌─────────────────┐          ┌──────────────────┐
│   App Cluster   │          │ Control Plane    │
│                 │          │                  │
│  ┌──────────┐   │  HTTPS   │  ┌───────────┐   │
│  │Konnector │◄──┼──────────┼──┤ Provider  │   │
│  │          │   │          │  │ APIs      │   │
│  └──────────┘   │          │  └───────────┘   │
│       │         │          │                  │
│       ▼         │          │                  │
│  APIService     │          │                  │
│  Bindings       │          │                  │
└─────────────────┘          └──────────────────┘
```

### Control Plane Mode (Konnector on Control Plane)

```
┌─────────────────┐          ┌──────────────────┐
│   App Cluster   │          │ Control Plane    │
│                 │          │                  │
│  Resources      │  HTTPS   │  ┌──────────┐    │
│                 │◄─────────┼──┤Konnector │    │
│                 │          │  └──────────┘    │
│                 │          │       │          │
│                 │          │       ▼          │
│                 │          │  APIService      │
│                 │          │  Bindings        │
│                 │          │       │          │
│                 │          │  ┌───┴───────┐   │
│                 │          │  │ Provider  │   │
│                 │          │  │ APIs      │   │
│                 │          │  └───────────┘   │
└─────────────────┘          └──────────────────┘
```

## Usage

### Prerequisites

1. A secret containing the app cluster kubeconfig must exist in the control plane cluster
2. The konnector must have RBAC permissions to read this secret
3. APIServiceBinding objects should be created in the control plane cluster

### App Cluster Requirements

The app cluster must allow the konnector identity to:

- Create/update CRDs used by bindings (bootstrap step)
- List/watch namespaces
- Create/update/list/watch resources for bound services in target namespaces

At minimum, the app cluster role needs:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: konnector-app-cluster
rules:
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["get","list","watch","create","update","patch"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get","list","watch"]
```

You may need to extend this role based on the APIs you bind (for example, to
create/read/update bound service CRs).

In default mode, the konnector typically runs with cluster-admin in the app cluster.
Control plane mode aims to scope the app cluster permissions to the minimum required.

### Control Plane Requirements

The control plane service account that runs the konnector must be able to:

- Read the app cluster kubeconfig secret
- List/watch APIServiceBindings and provider kubeconfig secrets

Example minimal RBAC (control plane cluster):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: konnector-control-plane
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get","list","watch"]
  - apiGroups: ["klutch.anynines.com"]
    resources: ["apiservicebindings"]
    verbs: ["get","list","watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: konnector-control-plane
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: konnector-control-plane
subjects:
  - kind: ServiceAccount
    name: konnector
    namespace: default
```

### Creating a Kubeconfig for Control Plane Mode

Kubeconfigs that rely on cloud auth exec plugins (for example, `aws eks get-token`)
do not work inside the control plane cluster. Use a service account token instead.

Example (app cluster):

```bash
# Create a dedicated service account
kubectl --kubeconfig=/tmp/app-cluster-kubeconfig create ns konnector-system
kubectl --kubeconfig=/tmp/app-cluster-kubeconfig create sa konnector -n konnector-system

# Bind the ClusterRole from the previous section
kubectl --kubeconfig=/tmp/app-cluster-kubeconfig apply -f - <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: konnector-app-cluster
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: konnector-app-cluster
subjects:
  - kind: ServiceAccount
    name: konnector
    namespace: konnector-system
EOF

# Create a kubeconfig that embeds the SA token
TOKEN=$(kubectl --kubeconfig=/tmp/app-cluster-kubeconfig create token konnector -n konnector-system)
SERVER=$(kubectl --kubeconfig=/tmp/app-cluster-kubeconfig config view --raw -o jsonpath='{.clusters[0].cluster.server}')
CA=$(kubectl --kubeconfig=/tmp/app-cluster-kubeconfig config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')

cat <<EOF >/tmp/app-cluster-kubeconfig-sa
apiVersion: v1
kind: Config
clusters:
- name: app
  cluster:
    server: ${SERVER}
    certificate-authority-data: ${CA}
contexts:
- name: app
  context:
    cluster: app
    user: konnector
    namespace: default
current-context: app
users:
- name: konnector
  user:
    token: ${TOKEN}
EOF
```

### Command Line Flags

Enable control plane mode with the following flags:

- `--control-plane-mode`: Enable control plane mode
- `--app-cluster-kubeconfig-secret-name`: Name of the secret containing the app cluster kubeconfig (required)
- `--app-cluster-kubeconfig-secret-namespace`: Namespace of the secret (required)
- `--app-cluster-kubeconfig-secret-key`: Key in the secret data containing the kubeconfig (default: "kubeconfig")

### Example

```bash
konnector \
  --control-plane-mode \
  --app-cluster-kubeconfig-secret-name=my-app-cluster \
  --app-cluster-kubeconfig-secret-namespace=klutch-system \
  --app-cluster-kubeconfig-secret-key=kubeconfig
```

### Deployment Example

Create a secret with the app cluster kubeconfig:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-app-cluster
  namespace: klutch-system
type: Opaque
stringData:
  kubeconfig: |
    apiVersion: v1
    kind: Config
    clusters:
    - cluster:
        certificate-authority-data: <base64-encoded-ca>
        server: https://app-cluster-api-server:6443
      name: app-cluster
    contexts:
    - context:
        cluster: app-cluster
        user: konnector
      name: app-cluster
    current-context: app-cluster
    users:
    - name: konnector
      user:
        token: <app-cluster-service-account-token>
```

Deploy the konnector on the control plane:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: konnector
  namespace: klutch-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: konnector
  template:
    metadata:
      labels:
        app: konnector
    spec:
      serviceAccountName: konnector
      containers:
      - name: konnector
        image: ghcr.io/anynines/klutchio/bind:latest
        command:
        - /konnector
        args:
        - --control-plane-mode
        - --app-cluster-kubeconfig-secret-name=my-app-cluster
        - --app-cluster-kubeconfig-secret-namespace=klutch-system
        - --lease-namespace=$(POD_NAMESPACE)
        env:
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
```

## Implementation Details

### Default Mode (konnector on app cluster)

1. **Initialization**: The konnector loads the app cluster kubeconfig (from in-cluster config or --kubeconfig flag)

2. **Resource Location**:
   - APIServiceBinding CRD: binding cluster (app cluster in this mode)
   - APIServiceBinding CRs: binding cluster (app cluster in this mode)
   - Bound service CRDs (e.g. `postgresqlinstances`): app cluster
   - Provider kubeconfig secrets: app cluster

3. **Informers**: The konnector sets up informers to:
   - Watch APIServiceBindings in the app cluster
   - Watch secrets (containing provider kubeconfigs) in the app cluster
   - Watch namespaces and CRDs in the app cluster

4. **Resource Syncing**: Resources are synchronized between provider APIs (from secrets) and the app cluster

### Control Plane Mode (konnector on control plane)

When control plane mode is enabled:

1. **Initialization**: The konnector loads the control plane kubeconfig (from in-cluster config or --kubeconfig flag) and fetches the app cluster kubeconfig from the specified secret

2. **Resource Location**:
   - APIServiceBinding CRD: binding cluster (control plane cluster in this mode)
   - APIServiceBinding CRs: binding cluster (control plane cluster in this mode)
   - Bound service CRDs (e.g. `postgresqlinstances`): **app cluster** (always installed where resources are synced)
   - Provider kubeconfig secrets: control plane cluster

3. **Informers**: The konnector sets up informers to:
   - Watch APIServiceBindings in the control plane
   - Watch secrets (containing provider kubeconfigs) in the control plane
   - Watch namespaces and CRDs in the app cluster

4. **Resource Syncing**: Resources are synchronized between provider APIs (from secrets) and the app cluster

## Security Considerations

- The app cluster kubeconfig secret should have restricted RBAC permissions (only the konnector should be able to read it)
- The service account token in the app cluster kubeconfig should have minimal required permissions
- Consider using short-lived tokens and implementing token rotation
- Network policies should restrict access to the control plane appropriately

## Troubleshooting

### Quick Checklist

- The app cluster kubeconfig secret exists and uses a service account token (no exec plugins)
- The konnector service account can `get` the kubeconfig secret in the control plane
- The app cluster service account can `get/list/watch` CRDs and namespaces
- The konnector deployment args reference the correct secret name/namespace/key

### Konnector fails to start with "failed to fetch app cluster kubeconfig secret"

- Verify the secret exists in the specified namespace
- Check that the konnector service account has permission to read the secret
- Ensure the secret contains the specified key

### Resources are not syncing

- Check that the app cluster kubeconfig is valid and the API server is reachable
- Verify that the service account in the app cluster kubeconfig has appropriate permissions
- Check konnector logs for connection errors

### APIServiceBindings not found

- **In default mode**: APIServiceBindings must be created in the app cluster
- **In control plane mode**: APIServiceBindings must be created in the control plane cluster, not the app cluster
- Verify you're creating the APIServiceBinding in the correct cluster based on your deployment mode
