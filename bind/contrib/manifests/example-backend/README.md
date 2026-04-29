# example-backend Deployment Modes

This directory contains deployment manifests for the example-backend in different modes.

## Modes

### Standard Mode (`insecure/`)

The default mode with full web server functionality:
- HTTP/HTTPS web server for OAuth2/OIDC authentication
- Consumer discovery via `/export` endpoint
- Resource selection via `/resources` endpoint
- Binding completion via `/bind` endpoint
- Session management with secure cookies
- All 5 controllers enabled

**Use when:**
- You need consumers to authenticate and bind directly
- You have external OIDC provider configured
- You're running the traditional klutch bind workflow

**Required secrets:**
- `oidc-config`: OIDC issuer credentials
- `cookie-config`: Cookie signing key

**Example deployment:**
```bash
kubectl apply -f insecure/example-backend.yaml
```

---

### Control Plane Mode (`control-plane-mode/`)

Lightweight mode for control plane-only operations:
- No HTTP/HTTPS web server
- No OIDC authentication
- No session/cookie management
- All 5 controllers enabled (including AppClusterBinding)
- Consumers must provide kubeconfig explicitly (via AppClusterBinding)
- Lower resource requirements

**Use when:**
- You're running controllers on the control plane cluster only
- Consumers already have kubeconfig or other authentication mechanism
- You want to minimize the footprint
- You're managing AppClusterBindings programmatically

**No secrets required**

**Example deployment:**
```bash
kubectl apply -f control-plane-mode/example-backend.yaml
```

---

## Key Differences

| Aspect | Standard | Control Plane Mode |
|--------|----------|-------------------|
| Web Server | ✅ Enabled | ❌ Disabled |
| OIDC Auth | ✅ Required | ❌ Not needed |
| Cookies | ✅ Required | ❌ Not needed |
| Controllers | ✅ All 5 | ✅ All 5 |
| Consumer Auth | OAuth2/OIDC | External (AppClusterBinding) |
| Resource Requirements | Higher | Lower |
| Secrets Required | oidc-config, cookie-config | None |

---

## Configuration

Both modes use the same binary but with different flags:

### Standard Mode
```bash
--namespace-prefix=cluster
--pretty-name=MangoDB
--consumer-scope=Namespaced
--oidc-issuer-client-id=<from secret>
--oidc-issuer-client-secret=<from secret>
--oidc-issuer-url=<from secret>
--oidc-callback-url=<from secret>
--listen-address=0.0.0.0:443
--cookie-signing-key=<from secret>
```

### Control Plane Mode
```bash
--namespace-prefix=cluster
--pretty-name=MangoDB
--consumer-scope=Namespaced
--control-plane-mode
```

The `--control-plane-mode` flag causes the application to:
- Skip web server initialization
- Skip OIDC provider setup
- Skip cookie handling
- Run only controller reconciliation loops

---

## AppClusterBinding Integration

The AppClusterBinding controller (enabled in both modes) allows programmatic management of app cluster connections:

```yaml
apiVersion: klutch.anynines.com/v1alpha1
kind: AppClusterBinding
metadata:
  name: my-app-cluster
spec:
  kubeconfigSecretRef:
    namespace: default
    name: app-cluster-kubeconfig
    key: kubeconfig
  apiExports:
    - group: database.example.com
      resource: postgresqls
    - group: cache.example.com
      resource: redisinstances
  konnector:
    deploy: true
    overrides:
      image: public.ecr.aws/w5n9a2g2/klutch/konnector:v1.4.0
```

This works in **both** modes:
- **Standard mode**: Alongside web-based authentication
- **Control plane mode**: As the primary integration mechanism
