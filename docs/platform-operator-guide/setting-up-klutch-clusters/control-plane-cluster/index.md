---
title: Setting Up the Klutch Control Plane Cluster
sidebar_position: 1
tags:
  - Klutch
  - control plane cluster
  - kubernetes
  - data services
  - platform operator
keywords:
  - Klutch
  - control plane cluster
  - kubernetes
  - data services
  - platform operator
---

To establish Klutch's multi-cluster architecture, it's essential to set up the Control Plane Cluster, which serves as
the central hub for managing data services.

This guide provides platform operators with step-by-step instructions to deploy and configure the Klutch Control Plane
Cluster.

## Prerequisites

Before setting up the Klutch Control Plane, ensure that the required infrastructure, tools, and dependencies are in place:

- **Kubernetes Cluster:** A running Kubernetes cluster is required to deploy the Control Plane components.
  - **Node Requirements:** If you plan to host highly available, container-based data services using the a8s framework
    within the Control Plane Cluster, ensure the cluster consists of at least three nodes. Each node should have
    specifications equivalent to or exceeding an AWS t3a.xlarge instance (4 vCPUs and 16 GiB memory).
- [Helm](https://helm.sh/docs/helm/helm_install/): Version 3.2.0 or later is required for package management.
- [kubectl](https://kubernetes.io/docs/tasks/tools/): The Kubernetes command-line tool must be installed and properly
configured to interact with your cluster, and include Kustomize support (built-in from v1.14+)
- [Crossplane®](https://docs.crossplane.io/latest/software/install/): Version 1.17.0 or newer must be installed on the
cluster.
  - Additionally, ensure the Server-Side Apply (SSA) flag is enabled for claims by setting:

      ```bash
      --set args='{"--enable-ssa-claims"}'
      ```

## Deploying the Control Plane

### 1. Install Klutch Components

Deploy the required Kubernetes objects for Klutch components to the Control Plane Cluster:

```bash
kubectl apply --kustomize https://github.com/anynines/klutchio
```

This command installs the necessary CustomResourceDefinitions (CRDs), composition functions and Crossplane® Providers
required for Klutch's operation, including:

- provider-anynines: Enables the provisioning and management of VM-based anynines data services in a Kubernetes-native
way.
- provider-kubernetes: Enables the provisioning and management of container-based data services using the anynines
a8s-framework.

Wait for the configuration package to become healthy:

```bash
kubectl get configuration -w
```

### 2. Configure Crossplane® Providers

To fully enable Klutch, at least one of the following providers must be configured:

1. provider-kubernetes (for container-based a8s data services)
2. provider-anynines (for VM-based a9s data services)
3. Both, if managing both types of services

#### 2.1 Configure provider-kubernetes (In-Cluster Provider Configuration)

The following command sets up a ProviderConfig resource for provider-kubernetes in Crossplane®. This configuration
defines how Crossplane® interacts with resources within the cluster:

```bash
kubectl apply -f https://raw.githubusercontent.com/anynines/klutchio/refs/heads/main/crossplane-api/deploy/config-in-cluster.yaml
```

This configuration uses **InjectedIdentity** as the credentials source, allowing Crossplane to authenticate and manage
in-cluster resources without requiring external credentials.

Check if the provider configuration has been successfully applied:

```bash
kubectl get providerconfigs.kubernetes.crossplane.io
```

#### 2.2 Configure provider-anynines for VM-Based a9s Data Services

If you plan to deploy VM-based data services using provider-anynines, you must configure access to the a9s Data Services
automation backend.

Obtain the credentials for a9s PostgreSQL, MariaDB, or other data services from anynines, then encode them in base64
format:

```bash
echo -n 'username' | base64
```

Replace `<data-service>` in the YAML file with the corresponding value from the table below.  Additionally, replace all
other placeholder values enclosed in `<>` with the corresponding actual values, following the specifications for each
Data Service you want to deploy:

| Data Service | Data-service Value |
| ------------ | ------------------ |
| a9s KeyValue | keyvalue |
| a9s Logme2 | logme2 |
| a9s MariaDB | mariadb |
| a9s Messaging | messaging |
| a9s MongoDB | mongodb |
| a9s PostgreSQL | postgresql |
| a9s Prometheus | prometheus |
| a9s Search | search |

The following YAML configuration sets up credentials and provider settings for the selected a9s Data Service. Ensure
base64-encoded values are used where required.

<details>
  <summary>Click to reveal the YAML configuration template for a9s Data Services</summary>

  ```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <data-service>-service-broker-creds
  namespace: crossplane-system
type: Opaque
data:
  username: <service-broker-username-base64-encoded>
  password: <service-broker-password-base64-encoded>
---
apiVersion: dataservices.anynines.com/v1
kind: ProviderConfig
metadata:
  name: <data-service>-service-broker
spec:
  url: <service-broker-url> # e.g. http://example.com:3000
  healthCheckEndpoint: "/osb_ext/v1/healthy"
  providerCredentials:
    source: Secret
    username:
      secretRef:
        namespace: crossplane-system
        name: <data-service>-service-broker-creds
        key: username
    password:
      secretRef:
        namespace: crossplane-system
        name: <data-service>-service-broker-creds
        key: password
---
apiVersion: v1
kind: Secret
metadata:
  name: <data-service>-backup-manager-creds
  namespace: crossplane-system
type: Opaque
data:
  username: <backup-manager-username-base64-encoded>
  password: <backup-manager-password-base64-encoded>
---
apiVersion: dataservices.anynines.com/v1
kind: ProviderConfig
metadata:
  name: <data-service>-backup-manager
spec:
  url: <backup-manager-url> # e.g. http://example.com:3000 or https://example.com:3001
  tls: #optional if tweaking is required
    # required if the name used to connect to the backup manager does not match the hostname
    # in the certificate. Set to the name in the certificate.
    overrideServerName: <cert-host-name> # e.g. example.com
    insecureSkipVerify: false # can be set to true to skip certificate validation
    # Can be used to manually configure certificate to be used. Skip if the
    # certificate is already valid within your org or globally.
    caBundleSecretRef: # ref to certificate chain in pem format.
      key: cert
      name: <data-service>-backup-manager-creds
      namespace: crossplane-system
  healthCheckEndpoint: "/v2/healthy" # Use "/instances" for a9s Backup Managers below v68
  providerCredentials:
    source: Secret
    username:
      secretRef:
        namespace: crossplane-system
        name: <data-service>-backup-manager-creds
        key: username
    password:
      secretRef:
        namespace: crossplane-system
        name: <data-service>-backup-manager-creds
        key: password
```

</details>

:::note

  The `healthCheckEndpoint` fields in the above `ProviderConfigs` default to `/instances` when not set. This endpoint can be inefficient and can cause service degradation when used. The alternative endpoints depend on whether the `ProviderConfig` is for a service broker or a backup manager. See the following table for information on which one to use.

  |Provider type|Recommended endpoint|Default endpoint|
  |-|-|-|
  |service broker|`/osb_ext/v1/healthy`|`/instances`|
  |backup manager|`/v2/healthy` in versions v68 and above|`/instances`|

:::

Repeat this process for each additional a9s Data Service you want to enable.

Once applied, check the provider configurations:

```bash
kubectl get providerconfigs.kubernetes.crossplane.io
```

### 3. Deploy klutch-bind Backend

The klutch-bind backend is a crucial component within the Klutch Control Plane Cluster, facilitating secure
communication between the Control Plane and App Clusters. For more information, please refer to the [architecture page](../../../architecture/index.md).

#### 3.1 Install a Certificate Manager (Prerequisite)

Before deploying klutch-bind, ensure that a certificate manager (such as cert-manager) is installed to handle TLS
certificates. If it is not installed, deploy it using:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

#### 3.2 Modify APIServiceExportTemplates

Navigate to the klutch-bind deployment directory:

```bash
cd /path/to/klutchio/bind/deploy/resources
```

Edit kustomization.yaml and uncomment the services you need (e.g., PostgreSQL). Then apply the modified Kustomization:

```bash
kubectl apply --kustomize /path/to/klutchio/bind/deploy/resources
```

### 4. Authentication Protocol Configuration

To manage user authentication, Klutch uses `OpenID Connect (OIDC)` for Single Sign-On (SSO).

To enable bindings, the OIDC backend must support the client credentials grant type. This setup requires an audience
mapper that adds klutch-bind as an audience in issued tokens. An audience mapper allows adding or modifying token
audiences to specify which applications or services can use them. You can find an example of how to set up OIDC with
Keycloak [here](./example-keycloak.md).

#### 4.1 Deploy the Backend

It is recommended to deploy the backend in the **bind** namespace, which was created during the Klutch [deployment](#deploying-the-control-plane).
Before applying the following YAML configuration, replace the placeholder values (`<>`) with actual values:

| Placeholder | Description |
| ----------- | ----------- |
| \<signing-key> | Cookie signing key. Generate with `openssl rand -base64 32.` |
| \<encryption-key> | Cookie encryption key. Generate with `openssl rand -base64 32.` |
| \<certificate> | Base64-encoded certificate of the Control Plane Cluster. Found in `clusters.certificate-authority-data` in the KubeConfig. |
| \<kubernetes-api-external-name> | External URL of the Kubernetes API server. Found in `clusters.server` in the KubeConfig. |
| \<oidc-issuer-client-url> | OIDC client URL, provided by your OIDC provider. |
| \<oidc-issuer-client-secret> | OIDC client secret, available in your OIDC provider's settings. |
| \<backend-host> | External address of the Klutch backend. Retrieve it from the LoadBalancer service using `kubectl get services -n bind` (value of `EXTERNAL-IP`). |
| \<add-your-email-here> | Email for Certificate Authority registration (e.g., Let's Encrypt via ACME). Update the Issuer in the YAML if using a different CA. |

**Deployment Considerations**

- **Ingress Controller:** The instructions assume the use of Nginx Ingress Controller. If using a different controller,
update `ingressClassName` in `networking.k8s.io/v1/Ingress` accordingly.
- **Certificate Authority:** This setup assumes Let's Encrypt CA with the ACME protocol. If using a different CA,
update `cert-manager.io/v1/Issuer` and `networking.k8s.io/v1/Ingress`.

Now, create a file named `backend.yaml` using the template below. Replace the placeholders with the appropriate values.

<details>
  <summary>Click to reveal the YAML configuration template for the backend</summary>

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: anynines-backend
  namespace: bind
  labels:
    app: anynines-backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: anynines-backend
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: anynines-backend
    spec:
      serviceAccountName: anynines-backend
      containers:
        - name: anynines-backend
          image: public.ecr.aws/w5n9a2g2/klutch/example-backend:v1.4.0
          args:
            - --namespace-prefix=cluster
            - --pretty-name=anynines
            - --consumer-scope=Namespaced
            - --oidc-issuer-client-id=$(OIDC-ISSUER-CLIENT-ID)
            - --oidc-issuer-client-secret=$(OIDC-ISSUER-CLIENT-SECRET)
            - --oidc-issuer-url=$(OIDC-ISSUER-URL)
            - --oidc-callback-url=$(OIDC-CALLBACK-URL)
            - --listen-address=0.0.0.0:9443
            - --cookie-signing-key=$(COOKIE-SIGNING-KEY)
            - --cookie-encryption-key=$(COOKIE-ENCRYPTION-KEY)
            - --external-address=<kubernetes-api-external-name>
            - --external-ca-file=/certa/ca
          env:
            - name: OIDC-ISSUER-CLIENT-ID
              valueFrom:
                secretKeyRef:
                  name: oidc-config
                  key: oidc-issuer-client-id
            - name: OIDC-ISSUER-CLIENT-SECRET
              valueFrom:
                secretKeyRef:
                  name: oidc-config
                  key: oidc-issuer-client-secret
            - name: OIDC-ISSUER-URL
              valueFrom:
                secretKeyRef:
                  name: oidc-config
                  key: oidc-issuer-url
            - name: OIDC-CALLBACK-URL
              valueFrom:
                secretKeyRef:
                  name: oidc-config
                  key: oidc-callback-url
            - name: COOKIE-SIGNING-KEY
              valueFrom:
                secretKeyRef:
                  name: cookie-config
                  key: signing-key
            - name: COOKIE-ENCRYPTION-KEY
              valueFrom:
                secretKeyRef:
                  name: cookie-config
                  key: encryption-key
          resources:
            limits:
              cpu: "2"
              memory: 2Gi
            requests:
              cpu: "100m"
              memory: 256Mi
          volumeMounts:
            - name: ca
              mountPath: /certa/
      volumes:
        - name: oidc-config
          secret:
            secretName: oidc-config
        - name: cookie-config
          secret:
            secretName: cookie-config
        - name: ca
          secret:
            secretName: k8sca
            items:
              - key: ca
                path: ca
---
apiVersion: v1
kind: Secret
metadata:
  name: cookie-config
  namespace: bind
type: Opaque
stringData:
  signing-key: "<signing-key>" # run: "openssl rand -base64 32"
  encryption-key: "<encryption-key>" # run: "openssl rand -base64 32"
---
apiVersion: v1
kind: Secret
metadata:
  name: k8sca
  namespace: bind
type: Opaque
stringData:
  ca: |
    <certificate>
---
apiVersion: v1
kind: Secret
metadata:
  name: oidc-config
  namespace: bind
type: Opaque
stringData:
  oidc-issuer-client-id: "klutch-bind"
  oidc-issuer-client-secret: "<oidc-issuer-client-secret>"
  oidc-issuer-url: "<oidc-issuer-client-url>"
  oidc-callback-url: "https://<backend-host>:443/callback"
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: anynines-backend
  namespace: bind
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: anynines-backend
  namespace: bind
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: anynines-backend
    namespace: bind
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: anynines-backend
  namespace: bind
  annotations:
    cert-manager.io/issuer: letsencrypt-prod # Adjust if not Let's Encrypt
spec:
  ingressClassName: nginx # Adjust if not Nginx Ingress Controller
  tls:
    - secretName: anynines-backend-tls
      hosts:
        - "<backend-host>"
  rules:
    - host: "<backend-host>"
      http:
        paths:
          - pathType: Prefix
            path: "/"
            backend:
              service:
                name: anynines-backend
                port:
                  number: 443
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt # Adjust if not  Let's Encrypt
  namespace: bind
spec:
  acme:
    # The ACME server URL
    server: https://acme-v02.api.letsencrypt.org/directory # Adjust if not Let's Encrypt
    email: <add-your-email-here>
    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt # Adjust if not  Let's Encrypt
    # Enable the HTTP-01 challenge provider
    solvers:
      - http01:
          ingress:
            class: nginx # Adjust if not Nginx Ingress Controller
```

</details>

After replacing the placeholder values, apply the file:

```bash
kubectl apply -f backend.yaml
```

## Next Steps

With the Klutch Control Plane Cluster successfully deployed, you're now ready to set up your App Cluster(s). Follow the
[App Cluster setup guide](../app-cluster.md) to deploy an App Cluster, provision data services, and start using them.
