# AppClusterBinding Controller

This controller manages AppClusterBinding resources on the control plane, automating app cluster management.

## Features

### 1. Kubeconfig Secret Validation
- Validates referenced secret exists and contains valid kubeconfig
- Verifies current context and namespace are properly configured
- Sets `SecretValid` condition on the AppClusterBinding status

### 2. APIServiceBinding Management
- Creates APIServiceBinding resources for each group/resource entry in `spec.apiExports`
- Labels managed bindings for ownership tracking:
  - `klutch.anynines.com/appclusterbinding-name`
  - `klutch.anynines.com/appclusterbinding-namespace`
- Updates existing bindings when spec changes
- Deletes bindings no longer in `spec.apiExports`

### 3. Konnector Deployment Automation
- Deploys konnector to **control plane cluster** when `spec.konnector.deploy` is true
- Runs in control plane mode with `--control-plane-mode` flag
- Configures konnector to connect to app cluster via kubeconfig secret reference
- Applies image override from `spec.konnector.overrides.image`
- Merges container settings from `spec.konnector.overrides.containerSettings` using strategic merge patch
- Creates/updates deployment in the AppClusterBinding's namespace on control plane
- Deployment name: `konnector-{binding-name}` to allow multiple per namespace
- Sets `KonnectorDeployed` condition on the AppClusterBinding status

## Example AppClusterBinding

```yaml
apiVersion: klutch.anynines.com/v1alpha1
kind: AppClusterBinding
metadata:
  name: my-app-cluster
  namespace: platform-ns
spec:
  kubeconfigSecretRef:
    namespace: platform-ns
    name: my-app-kubeconfig
    key: kubeconfig
  apiExports:
    - group: database.example.com
      resource: postgresqls
    - group: cache.example.com
      resource: redisinstances
    - group: messaging.example.com
      resource: rabbitmqs
  konnector:
    deploy: true
    overrides:
      image: public.ecr.aws/w5n9a2g2/klutch/konnector:v1.5.0
      containerSettings:
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Implementation Details

### Controller Startup
The controller is instantiated in [server.go](../../server.go) with:
- Dynamic informer for AppClusterBinding resources
- Secret indexer for efficient lookup by kubeconfig secret reference
- Bind client for APIServiceBinding CRUD operations

### Reconciliation Flow
1. **Validate Secret**: Check kubeconfig secret exists and is valid
2. **Manage Bindings**: Create/update/delete APIServiceBindings based on spec
3. **Deploy Konnector**: If enabled, deploy/update konnector in control plane with control-plane-mode flags

### Konnector Control Plane Mode
The konnector is deployed to the control plane cluster (where this controller runs) rather than the app cluster. It runs with these flags:
- `--control-plane-mode`: Enables control plane operation mode
- `--app-cluster-kubeconfig-secret-name`: Name of the kubeconfig secret
- `--app-cluster-kubeconfig-secret-namespace`: Namespace of the kubeconfig secret
- `--app-cluster-kubeconfig-secret-key`: Key within the secret (e.g., "kubeconfig")
- `--lease-namespace=$(POD_NAMESPACE)`: Namespace for leader election

This allows the konnector to manage APIServiceBindings on the control plane while syncing with the app cluster via the kubeconfig secret.

### Container Override Mechanism
The `containerSettings` field uses strategic merge patch to allow partial container spec overrides:
- Fields in `containerSettings` override defaults
- Unspecified fields retain default values
- Supports all Container spec fields (resources, env, volumeMounts, etc.)

## Conditions

### SecretValid
- **True**: Kubeconfig secret is valid and parseable
- **False**: Secret not found, missing key, or invalid kubeconfig

### KonnectorDeployed
- **True**: Konnector deployment created/updated successfully (or deploy=false)
- **False**: Failed to create app cluster client or deploy konnector

## Labels
Managed APIServiceBindings and konnector deployments are labeled with:
- `klutch.anynines.com/appclusterbinding-name`: Name of parent AppClusterBinding
- `klutch.anynines.com/appclusterbinding-namespace`: Namespace of parent AppClusterBinding
- `app`: "konnector" (for konnector deployments only)

These labels enable:
- Efficient lookup of managed resources
- Cleanup when apiExports list changes
- Ownership tracking
- Pod selection for konnector deployments
