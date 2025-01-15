# Crossplane API for Backing Service

A custom [Crossplane](https://github.com/crossplane/crossplane) API for
deploying anynines dataservices.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Prerequisites](#prerequisites)
3. [Optional: Build and push provider-anynines Package](#optional-build-and-push-provider-anynines-package)
4. [Optional: Build and push anynines-dataservices Package](#optional-build-and-push-anynines-dataservices-package)
5. [Installation](#installation)
6. [Usage - a8s](#usage---a8s)
7. [Usage - a9s](#usage---a9s)
8. [Update or Add a Service or Plan in a8s](#update-or-add-a-service-or-plan-in-a8s)

## Getting Started

These instructions will help you set up and run the project on your local
machine for development and testing purposes.

## Prerequisites

- Kubernetes cluster version v1.21.0 or later
  - You can deploy a Kubernetes cluster locally using tools such as
    [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) or [minikube](https://minikube.sigs.k8s.io/docs/start/)
- [Helm](https://helm.sh/docs/intro/install/) version v3.2.0 or later
- [Crossplane](https://docs.crossplane.io/v1.11/software/install/) version
1.17.1 or later.
  - You can deploy the *latest* version of Crossplane using helm by executing
    the following command:

    ```bash
    ./crossplane-api/deploy/deploy-crossplane.sh
    ```

  - Alternately, you can install a *specific* Crossplane version using helm's
    [*--version <version>*](https://docs.crossplane.io/v1.11/software/install/)
    option, and then deploy the [Crossplane Provider for Kubernetes](https://github.com/anynines/klutchio/blob/main/crossplane-api/deploy/provider-kubernetes.yaml).
- To build, push, and install Crossplane packages, you need the Crossplane CLI.
  You can install it with the following command:

    ```bash
    curl -sL https://raw.githubusercontent.com/crossplane/crossplane/master/install.sh | sh
    ```

## Optional: Build and push provider-anynines Package

### Create Provider and Provider Controller images for both amd64 and arm64 systems

```bash
cd provider-anynines
make build.all
```

```bash
#example
docker images
build-ee957d9f/provider-anynines-controller-amd64   latest    17d773ce8f47   8 seconds ago    46.9MB
build-ee957d9f/provider-anynines-controller-arm64   latest    5c2ea5ca73aa   26 seconds ago   45.7MB
build-ee957d9f/provider-anynines-amd64              latest    9742ae7e4cc1   32 seconds ago   92kB
build-ee957d9f/provider-anynines-arm64              latest    9742ae7e4cc1   32 seconds ago   92kB

```

To push the Crossplane Packages to AWS ECR, you need to retrieve an
authentication token and authenticate your Docker client to the registry. The
profile part is optional but useful if you have multiple accounts. To add the
access key on your machine run:

```bash
aws configure --profile=ECR
```

This step has to be performed only
once on each machine, after that, the aws cli will store your configuration.

If you have already configured an ECR profile, run:

```bash
export AWS_PROFILE=ECR
```

Then, to give Docker access to the registry, execute:

```bash
aws ecr-public get-login-password --region us-east-1 --profile=ECR | docker login --username AWS --password-stdin public.ecr.aws/w5n9a2g2
```

### Push images to ECR

There are two ECR repositories, one is used to store provider images
(`public.ecr.aws/w5n9a2g2/klutch/provider-anynines`) and the other
one is used for the provider controller images
(`public.ecr.aws/w5n9a2g2/klutch/provider-anynines-controller`).

> **Important Note!**
> To make sure you don't overwrite existing images, it's
important to use a unique image tag before pushing it to the ECR. To maintain
consistency and organization, it's recommended to follow the following format
for the image tags:

```bash
<NameInitials-Version> e.g. IM-v0.0.1
```

To push the controller images to ECR you should set the IMAGETAG variable:

```bash
make provider-controller-push IMAGETAG=<NameInitials-Version>

#example
make provider-controller-push IMAGETAG=IM-v0.0.1
```

While the controller images are built as two separate single-architecture images
they are both included in a multi-architecture manifest list as part of the
pushing procedure.

The manifest list is then tagged with with the image tag passed by the user without any modifications
while for the single-architecture images the tag passed by the user is supplemented with a suffix
signifying the architecture of the image (i.e. either `arm64` or `amd64`).

### Build and push provider package

To create a provider package using the existing provider
controller image and upload the package to the ECR image repository, you need to
execute the following command:

```bash
make provider-build-push IMAGETAG=<NameInitials-Version>

#example
make provider-build-push IMAGETAG=IM-v0.0.1
```

## Optional: Build and push anynines-dataservices Package

### Login to ECR

To push the Crossplane Packages to AWS ECR, you need to retrieve an
authentication token and authenticate your Docker client to the registry. The
profile part is optional but useful if you have multiple accounts. To add the
access key on your machine run:

```bash
aws configure --profile=ECR
```

This step has to be performed only
once on each machine, after that, the aws cli will store your configuration.

If you have already configured an ECR profile, run:

```bash
export AWS_PROFILE=ECR
```

Then, to give Docker access to the registry, execute:

```bash
aws ecr-public get-login-password --region us-east-1 --profile=ECR | docker login --username AWS --password-stdin public.ecr.aws/w5n9a2g2
```

### Build and push anynines-dataservices package

To create the dataservices Configuration Package we have to provide the version of the
configuration package we want to set.

```bash
make -C crossplane-api/ dataservices-config-push dataservicesConfigVersion=<image version>

# example
make -C crossplane-api/ dataservices-config-push dataservicesConfigVersion=v1.0.2
```

## Installation

### Install a8s control plane

Follow the instructions provided [here](https://github.com/anynines/a8s-deployment/blob/develop/docs/platform-operators/installing_framework.md#/install-the-a8s-control-plane)
to install the a8s control plane.

### Configure a9s data services

For development purposes, the provider is running on a local k8s cluster using
kind and access to the Service Broker and Backup Manager APIs is provided
through an SSH tunnel. Due to this setup, certain additional steps described
below are necessary.

#### Create ssh tunnel for Service Broker and Backup Manager

Follow the steps described [here](https://github.com/anynines/klutchio/blob/main/clients/a9s-backup-manager/README.md#pre-req) to get access to "aws-inception".

#### Start Service Broker tunnel

**NOTE:** *Instead of setting up the tunnel manually, you can also use `./scripts/create-instance-tunnel.sh`. For it's documentation, run the script without any parameters.*

In a separate terminal start an ssh tunnel to Service Broker

```bash
export SERVICE_INSTANCE_NAME=<dataservice name> #e.g. postgresql-ms-1686299661

```

**NOTE:** You would have to load the credentials to execute the bosh cli cmd in the BOSH director env.

```bash
export SERVICEBROKER_IP=$(ssh {IP of the virtual machine from where you can access the BOSH director};bosh -d $SERVICE_INSTANCE_NAME instances | grep broker/" | awk '{print $4}')

echo $SERVICEBROKER_IP

ssh -L 8989:$SERVICEBROKER_IP:3000 aws-s1-inception -o "ServerAliveInterval 30" -o "ServerAliveCountMax 3"
```

#### Start Backup Manager tunnel

In a separate terminal start an ssh tunnel to Backup Manager

```bash
export SERVICE_INSTANCE_NAME=<dataservice name> #e.g. postgresql-ms-1686299661
```

**NOTE:** You would have to load the credentials to execute the bosh cli cmd in the BOSH director env.

```bash
export BACKUP_MANAGER_IP=$(ssh {IP of the virtual machine from where you can access the BOSH director};bosh -d $SERVICE_INSTANCE_NAME instances | grep backup-manager/" | awk '{print $4}')

echo $BACKUP_MANAGER_IP

ssh -L 8988:$BACKUP_MANAGER_IP:3000 aws-s1-inception -o "ServerAliveInterval 30" -o "ServerAliveCountMax 3"
```

### Install crossplane providers

The whole package can either installed via kustomize or by manually applying each yaml file.

#### Kustomize

```bash
kubectl apply --kustomize ./crossplane-api/deploy
```

To check the status of the providers, use the following command:

```bash
kubectl get providers
```

Once the provider-kubernetes is healthy, configure it by applying:

```bash
kubectl apply -f crossplane-api/deploy/config-in-cluster.yaml
```

#### Manual

Install provider-kubernetes for a8s Data Services, currently supporting a8s
PostgreSQL.

```bash
kubectl apply -f crossplane-api/deploy/provider-kubernetes.yaml
```

To check the status of the provider, use the following command:

```bash
kubectl get providers
```

Once the provider-kubernetes is healthy, configure it:

```bash
kubectl apply -f crossplane-api/deploy/config-in-cluster.yaml
```

Next install the provider-anynines:

```bash
kubectl apply -f crossplane-api/deploy/provider-anynines.yaml
```

To install the configuration package (containing definitions and compositions), there are two options:

1. Install the package via crossplane:

```bash
crossplane xpkg install configuration public.ecr.aws/w5n9a2g2/klutch/dataservices:v1.3.1
```

2. Install files directly:

```bash
kubectl create --kustomize crossplane-api
```

To check the configuration and providers' status, use the following commands:

```bash
kubectl get configuration
kubectl get providers
```

> **Note**
> When deploying provider-anynines in a cluster, Crossplane dynamically
manages RBAC. However, if provider-anynines is not installed in the cluster,
Crossplane won't be able to manage RBAC dynamically. As a result, Compositions
will not be able to configure the provider-anynines managed resources due to
authorization issues.

### Install Crossplane Functions

Additionally, we install composition functions. Composition functions (or simply “functions”) are Crossplane extensions that template Crossplane resources. Crossplane uses these functions to determine which resources to create when a composite resource (XR) is created. To verify that the composition functions are correctly installed, use the following command:

```bash
kubectl get function
```

Expected output:

```text
NAME                           INSTALLED   HEALTHY   PACKAGE                                                                  AGE
function-patch-and-transform   True        True      xpkg.upbound.io/crossplane-contrib/function-patch-and-transform:v0.1.4
```

#### Install ProviderConfig for provider-anynines

To configure the provider, based on your development environment, make sure to
install the appropriate ProviderConfig.

You can use the provided Makefile to effortlessly retrieve the necessary
credentials, automatically populate Secrets, and create the appropriate
providerConfigs for your chosen data service(s). Simply provide the type and
name of the data service instance you are using.

For example, you can create and apply providerConfig for PostgreSQL
instances using the following command:

```bash
make -C crossplane-api/ providerconfig postgresInstanceName=<postgres_instance_name> searchInstanceName=<search_instance_name> mongodbInstanceName=<mongodb_instance_name> logme2InstanceName=<logme2_instance_name> messagingInstanceName=<messaging_instance_name> mariadbInstanceName=<mariadb_instance_name> prometheusInstanceName=<prometheus_instance_name>
```

The providerConfigs applied this way assume that you are running the provider-anynines in a local
kind cluster and have set up SSH tunneling to have access to an a9s Service Broker. Therefore they
have their `spec.url` field set to `http://dockerhost:[port number for their service]` (see
[Port selection](#port-selection) for a table detailing which data service is mapped to which port).

If you want to deploy the provider-anynines in an EKS cluster that is either in the same network as
the a9s Service Brokers of the inception staging environment or that has a VPC peering to that
environment, then you can use the following command to generate a providerConfig where `spec.url`
has the IP and port number under which the specified Service Broker is available in the network:

```bash
make -C crossplane-api/ providerconfig postgresInstanceName=<postgres_instance_name> GET_BROKER_IP="true"
```

Currently this only works with providerConfigs for PostgreSQL (setting `GET_BROKER_IP` to `"true"`
for other data services has no effect) but we plan on adding support for this in other data services
once we start End-to-End testing them.

#### Port selection

We offer default [ProviderConfigs](/provider-anynines/examples/provider/)
designed to adhere to a port selection convention when operating locally via
port forwarding. The prescribed convention is as follows:

- PostgreSQL: `8989` and `8988` for `Service Broker` and `Backup Manager` respectively.
- Search: `9189` and `9188` for `Service Broker` and `Backup Manager` respectively.
- MongoDB: `9389` and `9388` for `Service Broker` and `Backup Manager` respectively.
- Logme2: `9489` and `9488` for `Service Broker` and `Backup Manager` respectively.
- MariaDB: `9589` and `9588` for `Service Broker` and `Backup Manager` respectively.
- Messaging: `9689` and `9688` for `Service Broker` and `Backup Manager` respectively.
- Prometheus: `9789` and `9788` for `Service Broker` and `Backup Manager` respectively.

### Create a k8s Service to access localhost

Since the Service Broker and Backup Manager APIs are accessible through an SSH
tunnel, we will deploy a k8s Service within the local kind cluster to enable
communication between the pods and the "localhost" on our development machine.
To deploy this service, simply execute the provided code:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: dockerhost
  namespace: crossplane-system
spec:
  type: ExternalName
  externalName: host.docker.internal
EOF
```

## Usage - a8s

### Provision an a8s PostgreSQL instance

Adjust the [Composite Resource Claim](examples/a8s/postgresql-claim.yaml)
to refer to a valid service and plan.

| Service          | Version       |
|------------------|---------------|
| a9s-postgresql13 | PostgreSQL 13 |
| a9s-postgresql14 | PostgreSQL 14 |

| Plan                      | Replicas| Volume Size | CPU | Memory |
|---------------------------|---------|-------------|-----|--------|
| postgresql-single-nano    | 1       | 3Gi         | 2   | 1 Gi   |
| postgresql-single-small   | 1       | 10Gi        | 2   | 2 Gi   |
| postgresql-single-medium  | 1       | 50Gi        | 2   | 4 Gi   |
| postgresql-single-big   | 1       | 200Gi       | 4   | 16 Gi  |
| postgresql-replicas-small  | 3       | 10Gi        | 2   | 2 Gi   |
| postgresql-replicas-medium | 3       | 50Gi        | 2   | 4 Gi   |
| postgresql-replicas-big  | 3       | 200Gi       | 4   | 16 Gi  |

```bash
kubectl apply -f ./crossplane-api/examples/a8s/postgresql-claim.yaml
```

### Create a8s ServiceBinding

The servicebinding claim must target an existing PostgreSQL claim name.

```bash
kubectl apply -f ./crossplane-api/examples/a8s/servicebinding-claim.yaml
```

> **Note**
> `ServiceBindings` are not designed to target a data services within arbitrary `Namespaces`. This design restriction enables `ServiceBindings` to work within the `kube-bind` context, where `Namespaces` are dynamically allocated.

### Create Backup

The backup claim must target an existing PostgreSQL claim name.

```bash
kubectl apply -f ./crossplane-api/examples/a8s/backup-claim.yaml
```

### Create a8s Restore

The restore claim must target an existing PostgreSQL claim name.

```bash
kubectl apply -f ./crossplane-api/examples/a8s/restore-claim.yaml
```

## Usage - a9s

### Provision an a9s service instance

You should create a service instance by using a supported service and plan.
To access the catalog that includes all the supported services and plans, modify
the following command by replacing ```<user>``` and ```<password>``` with the
appropriate Service Broker credentials and the ```<port>``` with the local port
through which the Service Broker is available locally via the SSH tunnel.
Then, execute the following command:

```bash
curl http://<user>:<password>@localhost:<port>/v2/catalog -H "X-Broker-API-Version: 2.14" | jq
```

Under the `crossplane-api/examples/a9s` folder, you'll find examples for various
service instances. For instance, you can deploy PostgreSQL by applying the
PostgreSQL Composite Resource Claim (XRC) using the following command:

```bash
kubectl apply -f ./crossplane-api/examples/a9s/postgresql/serviceinstance-claim.yaml
```

You can check it with:

```bash
kubectl get postgresqlinstances.anynines.com
NAME                          MANAGEDRESOURCE   SYNCED   READY   CONNECTION-SECRET   AGE
example-postgresql-instance   provisioned       True     True                        4m19s
```

And its managed resource with:

```bash
kubectl get serviceinstances.dataservices.anynines.com
```

### Create a9s ServiceBinding

The servicebinding claim must target an existing service instance. For example,
you can create a ServiceBinding to a PostgreSQL instance with the following
command:

```bash
kubectl apply -f ./crossplane-api/examples/a9s/postgresql/servicebinding-claim.yaml
```

### Create a9s Backup

The backup claim must target an existing service instance. For example, you can
create a Backup from a PostgreSQL instance with the following command:

```bash
kubectl apply -f ./crossplane-api/examples/a9s/postgresql/backup-claim.yaml
```

### Create a9s Restore

The restore claim must target an existing service instance backup. For example,
you can restore a PostgreSQL Backup with the following command:

```bash
kubectl apply -f ./crossplane-api/examples/a9s/postgresql/restore-claim.yaml
```

## Update or Add a Service or Plan in a8s

In case of a Service or Plan is changed or a new one is added, it is essential
to update both the [composition](api/a8s/postgresql/composition.yaml)
and [definition](api/a8s/postgresql/definition.yaml) yaml files.
In our current approach, we validate the user-defined Service and Plan and
translate Service and Plan names from the Composite Resource (XR) to data
service version, CPU, memory and volume sizes in the Managed Resource (MR). If a
new Service or Plan is added, it is necessary to update the corresponding
composition and definition files to enable XR to MR translation and validation.
In case of any modifications to values for CPU, memory, and volume size of an
existing Plan, the corresponding hard-coded values in composition file should
also be updated accordingly. Please ensure you update these 2 files to
accurately reflect the updated/new Service or Plan, allowing users to
effectively provision the desired resources.

Here is a breakdown of the steps to follow for different scenarios:

### Update a Plan in a8s

1. Edit the [composition.yaml](api/a8s/postgresql/composition.yaml)
file using a text editor or IDE.

2. Locate the [metadata.labels](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/a8s/postgresql/composition.yaml#LL5C3-L5C10)
field at the top of the file. Within this field, you can see the name and the
corresponding value for disk, cpu and memory resources for the different Plan
sizes.

    For example, for medium sized Plans (e.g. postgresql-single-medium) we have:

    ```yaml
    volumeSizeMedium: &volumeSizeMedium "50Gi"
    CPUMedium: &CPUMedium "2"
    MemoryMedium: &MemoryMedium "4Gi"
    ```

3. Modify the necessary fields to reflect the changes in the Plan.

    For example, suppose that we want to update the volumeSize of the medium
    sized Plan from 50Gi to 75Gi.
    In this case we would change the:

    ```yaml
    volumeSizeMedium: &volumeSizeMedium "50Gi"
    ```

    to:

    ```yaml
    volumeSizeMedium: &volumeSizeMedium "75Gi"
    ```

4. Save the updated composition.yaml file.

### Add a Plan

1. Edit the [definition.yaml](api/common/postgresql_definition.yaml)
and [composition.yaml](api/a8s/postgresql/composition.yaml)
files using a text editor or IDE.

2. Locate the [supported.plans](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/common/postgresql_definition.yaml#L49)
field in [definition.yaml](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/common/postgresql_definition.yaml).
Within this field you can see a list of supported Plans:

    ```yaml
    plans: &pgPlans ["postgresql-replicas-small",
        "postgresql-replicas-medium", "postgresql-replicas-big",
        "postgresql-single-nano","postgresql-single-small",
        "postgresql-single-medium", "postgresql-single-big"]
    ```

3. Update the "plans" list with the new Plan to be supported.

    For example, suppose the new plan "postgresql-single-huge" is
    introduced, so the list will be updated to:

    ```yaml
    plans: &pgPlans ["postgresql-replicas-small",
        "postgresql-replicas-medium", "postgresql-replicas-big",
        "postgresql-single-nano","postgresql-single-small",
        "postgresql-single-medium", "postgresql-single-big",
        "postgresql-single-huge"]
    ```

4. Update the validation rules.

    Because only upgrades from smaller to larger DS instance sizes are allowed,
    the validation should also be updated. The validation rule is located in the
    [definition.yaml](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/common/postgresql_definition.yaml#L30)
    file under the x-kubernetes-validations.rule field.

    Continuing the example with the "postgresql-single-huge", the
    validation in this case should be updated with the following rules that
    prohibit the transition from extralarge to smaller dataservice instances.

    ```yaml
    !(self.find('[A-Za-z]+$') == 'small' && oldSelf.find('[A-Za-z]+$') == 'extralarge')
    !(self.find('[A-Za-z]+$') == 'medium' && oldSelf.find('[A-Za-z]+$') == 'extralarge')
    !(self.find('[A-Za-z]+$') == ‘large’ && oldSelf.find('[A-Za-z]+$') == ‘extralarge’)
    ```

5. Additionally, the [metadata.labels](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/a8s/postgresql/composition.yaml#L5)
    field in the [composition file](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/a8s/postgresql/composition.yaml)
    should also be updated.

    For the "postgresql-single-huge" example, we could add something
    similar to:

    ```yaml
    volumeSizeHuge: &volumeSizeHuge "1000Gi"
    CPUHuge: &CPUHuge "8"
    MemoryHuge: &MemoryHuge "32Gi"
    ```

6. Finally, the [maps](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/a8s/postgresql/composition.yaml#L53)
    used for patching the disk, cpu and memory resources in the
    [composition file](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/a8s/postgresql/composition.yaml)
    should also be updated.

    For our favorite "postgresql-single-huge" example this could mean
    adding to the maps something like:

    ```yaml
    ...
    - type: map
        map:
            nano: *volumeSizeNano
            small: *volumeSizeSmall
            medium: *volumeSizeMedium
            big: *volumeSizeLarge
            huge: *volumeSizeHuge

    ...
    - type: map
        map:
            nano: *CPUNano
            small: *CPUSmall
            medium: *CPUMedium
            big: *CPULarge
            huge: *CPUHuge

    ...
    - type: map
        map:
            nano: *MemoryNano
            small: *MemorySmall
            medium: *MemoryMedium
            big: *MemoryLarge
            huge: *MemoryHuge
    ...
    ```

7. Save the updated composition.yaml and definition.yaml files.

### Add/Update a Service

In our current implementation we extract the dataservice version from the
service name. Consequently, updating a service would mean deleting the current
service and creating a new one. Thus, in this case, a Service update is
equivalent to adding a new Service. The steps to add a Service are:

1. Open the [definition.yaml](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/common/postgresql_definition.yaml)
file using a text editor or IDE.

2. Locate the [supported.services](https://github.com/anynines/klutchio/blob/main/crossplane-api/api/common/postgresql_definition.yaml#L34)
field.
Within this field you can see a list with the supported Services:

    ```yaml
    services: &pgServices ["a9s-postgresql13","a9s-postgresql14"]
    ```

3. Update the "services" list with the new Service to be supported.

    For example, suppose the new Service "a9s-postgresql15" is introduced, so
    the list will be updated to:

    ```yaml
    services: &pgServices ["a9s-postgresql13","a9s-postgresql14", "a9s-postgresql15"]
    ```

4. Save the updated definition.yaml file.

### **Important note!**

By default, when a Composition is updated or created, all XRs that use said
Composition will be updated instantaneously. This means that any changes
made to the Composition, even minor ones like label and annotation
modifications, by the platform team would instantly affect the XRs provisioned
and owned by app teams.

Each time a Composition is updated, a CompositionRevision is automatically
created by a controller. The schema of the CompositionRevision encompasses the
schema of the Composition itself. This mechanism allows an XR instance to be
associated with a particular revision of a Composition, effectively "pinning" it
to that specific version. As a result, updates made to the Composition will not
inherently impact the XR instances that (indirectly) use it.

By default, an XR instance will have the optional filed "compositionUpdatePolicy"
set to "Automatic" causing it to always select the latest revision of the
desired Composition. Only when the compositionUpdatePolicy of a previously
created XR(C) is set to "Manual" its behavior deviates from the default and
requires manual updates.

Therefore, It is important to carefully consider the approach that minimizes
unexpected outcomes for users. Platform operators should exercise caution when
updating Compositions and prioritize creating and testing new Compositions
before deploying them.

### Build configuration images

A Crossplane Configuration image serves as an alternative approach to directly
applying the XRDs and Composition yaml files. This approach allows for easier
management, versioning, and distribution of the configuration changes.
To incorporate the changes from the updated XRDs and Compositions to a Crossplane
Configuration Package you have to build and push a new Configuration image.
A guide to creating Crossplane Configuration Packages can be found
[here](https://anynines.atlassian.net/wiki/spaces/A8S/pages/2859466893/Building+Crossplane+Configuration+Package+images+with+the+Crossplane+CLI).
