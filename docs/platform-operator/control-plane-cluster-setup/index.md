---
id: klutch-po-data-services
title: Setting up Control Plane and App Clusters
tags:
  - control plane cluster
  - kubernetes
  - data services
  - platform operator
keywords:
  - data services
  - platform operator
---

Below are the instructions for setting up Klutch's Control Plane Cluster, which includes
deploying [Crossplane](https://www.crossplane.io/). While other cloud providers are supported as
well, for the purpose of this example, we will use [AWS](https://aws.amazon.com/). These
instructions also cover the configuration of the App Cluster - i.e. the cluster from which data
services will be used - with bindings to services exported in the Control Plane Cluster.

## Prerequisites

- Provision an EKS cluster
  - Use a minimum of 3 nodes if you want to host highly available services on the Control Plane
    Cluster, each node should at least be t3a.xlarge or equivalent.
  - In general, the Klutch control plane itself can also run with just one worker node.
- Set up a VPC with 3 subnets.
- Make sure [eksctl](https://eksctl.io/introduction/#getting-started) is installed and configured
  correctly.

## Overview

To successfully manage data services using Klutch, several components must be deployed. Konnector is
deployed on each App Cluster that wants to manage its data services with Klutch. Klutch itself
is deployed on a Control Plane Cluster. Konnector is configured to correctly interact with
klutch-bind running in Klutch so each service running on the App Cluster doesn't need to be
configured to call Klutch. Instead, the services can use Klutch to manage their data services by
interacting with Konnector.

![Deploy Klutch and its related components](klutch-deployment.png)

The following instructions will install the services that are necessary to use Klutch. First, the
Crossplane provider `provider-anynines` is installed in the Control Plane Cluster. This is done by
installing both the provider itself and configuration that the provider needs to run properly.

Then, the klutch-bind backend is deployed in the Control Plane Cluster. The installation for
klutch-bind includes permission configuration that needs to be set up so the App Cluster can
properly access the backend.

Lastly, Konnector must be [installed on the App Cluster](./setup-app-cluster.md). After
installation, Konnector is bound to the klutch-bind backend. This is how the App Cluster can call
Klutch in the Control Plane Cluster.

The current instructions only include deployment of `provider-anynines`. This product is currently
in development and more providers can be expected soon!

## Setup Klutch Control Plane Cluster

We will now set up the Kubernetes Control Plane Cluster that you've set up in the previous step
so that we can deploy Klutch on it.

### Deploy Crossplane and provider-anynines

#### Prerequisites

- [Helm](https://helm.sh/docs/intro/install/) version v3.2.0 or later
- [Crossplane](https://docs.crossplane.io/latest/software/install/) version 1.15.0 or newer must be
  installed on the cluster.
  - Additionally, ensure the Server-Side-Apply flag is enabled for claims by setting
    `--set args='{"--enable-ssa-claims"}'`.
- Install the [Crossplane CLI](https://docs.crossplane.io/latest/cli/)

#### Installation of Klutch

You can install Klutch by following the given instructions:

1. Install Klutch components by executing:

   ```bash
   kubectl apply --kustomize https://github.com/anynines/klutchio
   ```

2. Wait for the providers to become healthy:

   ```bash
   kubectl get providers -w
   ```

3. Wait till the configuration package state is healthy before moving on to the next steps:

   ```bash
   kubectl get configuration -w
   ```

4. Now add configuration for `provider-kubernetes` by executing the following command:

   ```bash
   kubectl apply -f https://github.com/anynines/klutchio/blob/main/crossplane-api/deploy/config-in-cluster.yaml
   ```

### Deploy klutch-bind

#### Prerequisites

- [cert-manager](https://cert-manager.io/docs/installation/) - for this installation we are using
  cert-manager but you can freely choose your own certificate managers.

#### Deployment

The following command deploys Klutch to the (newly created) namespace `bind`:

```sh
kubectl apply --kustomize https://github.com/anynines/klutchio/bind/deploy/resources
```

:::info

Note that you can clone the repository and modify the
[Kustomize file](https://github.com/anynines/klutchio/tree/main/bind/deploy/resources/kustomization.yaml)
to support the desired data services.

:::

#### Authentication protocol configuration

We need an authentication protocol to manage user verification securely.
[OpenID Connect (OIDC)](https://openid.net/developers/how-connect-works/) is an example of such a
protocol, providing a standardized method for user authentication and profile information retrieval.
We've therefore adopted the OIDC method for enabling single sign-on (SSO) into our Kubernetes
cluster. We require an "audience mapper" that adds the audience `klutch-bind` to tokens issued. In a
wider context, an audience mapper allows you to add or modify audiences (applications or services)
intended to use a token. To enable bindings without user interaction, for example created by a
script or other automation the OIDC backend needs to support the
[client credentials grant type](https://datatracker.ietf.org/doc/html/rfc6749#section-4.4).

You can find an example of how to set up OIDC using [KeyCloak](./oidc-keycloak).

#### Deploy the backend

:::note

We recommend using the namespace `bind` for the deployment of the backend. This namespace was
created when you [deployed Klutch](#deployment).

:::

In the [YAML file that describes the deployment of the backend](#deploy-the-klutch-backend), make
sure to replace the placeholder values indicated by `<>` with their corresponding actual values. The
values that require updating include:

| Placeholder                      | Description                                                                   |
| -------------------------------- | ----------------------------------------------------------------------------- |
| `<signing-key>`                  | Cookies signing key - run `openssl rand -base64 32` to create it              |
| `<encryption-key>`               | Cookies encryption key - run `openssl rand -base64 32` to create it           |
| `<certificate>`                  | The base64 encoded certificate of the Control Plane Cluster   |
| `<kubernetes-api-external-name>` | URL of the Kubernetes API server of the Control Plane Cluster |
| `<oidc-issuer-client-url>`       | OIDC client url                                                               |
| `<oidc-issuer-client-secret>`    | OIDC client secret                                                            |
| `<backend-host>`                 | the URL of the Klutch backend service, see [backend-host](#backend-host)      |
| `<add-your-email-here>`          | Email address used for Certificate Authority registration                     |

##### Signing and Encryption keys generation

Signing and encryption keys can be generated using the following command:

```bash
openssl rand -base64 32
```

##### OIDC Credentials

Your OIDC provider will make available the client URL and client secret used for
`<oidc-issuer-client-id>` and `<oidc-issuer-client-secret>`. You can locate these values within the
settings or configuration section of your chosen OIDC provider. Instructions for setting up OIDC
using KeyCloak can be found [here](./oidc-keycloak).

##### ACME email address

The email specified in `<add-your-email-here>` should be the email used for registering with a
certificate authority (CA), such as Let's Encrypt. This guide suggests using Let's Encrypt with the
ACME protocol. If a different approach is preferred, please update the `Issuer` in the provided
[YAML manifest](#deploy-the-klutch-backend).

#### Kubernetes cluster certificate

The base64 encoded certificate of the Control Plane Cluster can be found in the
KubeConfig of that cluster, specifically in `clusters.certificate-authority-data`.

#### Kubernetes api external name

The URL of the Kubernetes API server of the Control Plane Cluster, .i.e. the
Kubernetes API server's external hostname can be found in kubeConfig `clusters.server`.

#### backend-host

During the [deployment of Klutch](#deployment) a service of type `LoadBalancer` was created. This
load balancer can be used to connect to Klutch from an App Cluster. To obtain the
required information about the service, execute the following command:

```bash
$ kubectl get services -n bind
NAME               TYPE           CLUSTER-IP    EXTERNAL-IP               PORT(S)         AGE
anynines-backend   LoadBalancer   10.10.10.10   something.amazonaws.com   443:32686/TCP   6m29s
```

Use the value of `EXTERNAL-IP` (which may be a hostname instead of an IP address) as `backend-host`.

#### Deploy the Klutch backend

Then apply the following YAML file within the `bind` namespace. This action will deploy the
anynines-backend Deployment, Services, Ingress, Configmaps and Secrets for klutch-bind.

The instructions assume the usage of the
[Nginx Ingress Controller](https://kubernetes.github.io/ingress-nginx/). If a different Ingress
controller is used, please adjust the `ingressClassName` value in `networking.k8s.io/v1/Ingress` to
match your Ingress controller in the following YAML manifest. Additionally, modify the
`cert-manager.io/v1/Issuer` if needed.

Moreover, this setup assumes the use of Let's Encrypt CA with ACME protocol. Adjust
`cert-manager.io/v1/Issuer` and `networking.k8s.io/v1/Ingress` in the following YAML file if you are
using a different CA.

The following YAML text describes the Kubernetes manifest for Klutch's backend (see below for a link
the allows you to download this file):

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
          image: public.ecr.aws/w5n9a2g2/anynines/kubebind-backend:v1.3.0
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

<a href="/po_files/backend-anynines.template.yaml" target="_blank" download>Download
backend-anynines.template.yaml</a>

After downloading the `backend-anynines.template.yaml` file, replace the indicated placeholder
values denoted by `<>` and then execute the following command to apply the file:

```bash
cp backend-anynines.template.yaml backend-anynines.yaml
... edit backend-anynines.yaml ...
kubectl apply -f backend-anynines.yaml
```

## Coming soon

Platform operators will soon have access to new Kubernetes-integrated features for managing data
services, including configuration options for disk and memory usage, streamlined updates, and robust
disaster recovery. Additionally, platform operators will be able to monitor and log services by
collecting metrics and retrieving logs.
