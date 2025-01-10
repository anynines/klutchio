# Developer Guide

## Setup

Create a Kind cluster for testing.

```bash
kind create cluster --name=provider-anynines
```

Use alias for kubectl

```bash
alias k=kubectl
```

Using [provider-anynines](https://github.com/anynines/klutchio/tree/main/provider-anynines) execute the following commands.

Install CRDs

```bash
k apply -f package/crds
```

Create crossplane-system namespace

```bash
k create ns crossplane-system
```

Use [Setup tunnel to service-broker](https://anynines.atlassian.net/browse/A8S-1205) to configure port-forward to anynines service-broker and backup-manager.

Update `examples/provider/config.yaml` to use the username and password for the service-broker and backup-manager derived using the scripts from [Setup tunnel to service-broker](https://anynines.atlassian.net/browse/A8S-1205). URL should be `http://localhost:8989`.

The backup-manager is the same procedure. Ensure that you use a separate port.

For example:

```bash
export SERVICE_INSTANCE_NAME=postgresql-ms-1686299661 #e.g. postgresql-ms-1686299661

export BACKUP_MANAGER_IP=$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $SERVICE_INSTANCE_NAME instances | grep backup-manager" | awk '{print $4}')

echo $BACKUP_MANAGER_IP

ssh -L 8988:$BACKUP_MANAGER_IP:3000 aws-s1-inception -o "ServerAliveInterval 30" -o "ServerAliveCountMax 3"
```

Edit `examples/provider/config.yaml` in the same way as the service broker. URL should be `http://localhost:8988`.

Once you have updated the config file, apply it,

```bash
k apply -f examples/provider/config.yaml
```

Fetch the catalog from the service-broker so that we can provide the correct UUIDs for Service and Plan that the service-broker expects.

```bash
curl http://<USERNAME>:<PASSWORD>@localhost:8989/v2/catalog -H "X-Broker-API-Version: 2.14" | jq
```

Configure the ServiceInstance MR located at `examples/sample/postgresql-example.yaml` to look like this:

```bash
apiVersion: dataservices.anynines.com/v1
kind: ServiceInstance
metadata:
  name: example-pg-instance-jklasd
  labels:
    crossplane.io/claim-name: example-pg-instance
    crossplane.io/claim-namespace: default
spec:
  forProvider:
    acceptsIncomplete: true
    # when testing with inception a suffix consisting of a '-' and the
    # name of the deployment must be appended to the service name
    # e.g. a9s-postgresql10-ms-1686387226
    serviceName: a9s-postgresql10
    planName: postgresql-single-small
    organizationGuid: a1d46b5c-b639-4f43-85c7-e9a0e5f01f75
    spaceGuid: 1bf71cf3-9017-4846-bffc-b9b31872bfaf
  providerConfigRef:
    name: anynines-service-broker
```

## Create and Use Service Instance

Apply the Managed Resource for anynines PostgreSQL.

```bash
kubectl apply -f examples/sample/postgresql-example.yaml  -o yaml
```

Wait for the resource to transition from `state: deploying` to `state: provisioned` by looking at the Status field of the resource.

```bash
watch kubectl apply -f examples/sample/postgresql-example.yaml  -o yaml
```

Once the PostgreSQL instance is provisioned we need to create a service-binding.

```bash
watch kubectl apply -f examples/sample/sb-example.yaml  -o yaml
```

provider-anynines will create a secret for you named `example-sb-creds` in the `crossplane-system` namespace which you can use to access the instance. The Inception test environment doesn't expose the instances to the outside so we need to SSH onto Inception to interact with the database.

```bash
watch kubectl apply -f examples/sample/sb-example.yaml  -o yaml
```

ssh onto Inception and then onto the service broker. The service-broker can be accessed using `bosh -d <SERVICE-BROKER-NAME> ssh broker`. Then install the postgresql-client using `sudo apt update && sudo apt install -y postgresql-client`.

Use this script to fetch and decode the ENDPOINT.

```bash
#!/bin/bash

# Set Kubernetes context and namespace
NAMESPACE="crossplane-system"

# Name of the secret
SECRET_NAME="example-sb-creds"

# Fetch secret
SECRET=$(kubectl -n "$NAMESPACE" get secret "$SECRET_NAME" -o json)

# Extract, decode data and create PostgreSQL URI
ENDPOINT=$(echo $SECRET | jq -r '.data.endpoint' | base64 -d)

# Construct PostgreSQL URI
PSQL_URI=$ENDPOINT

echo "PostgreSQL URI: ${PSQL_URI}"
```

Now you can access the PostgreSQL server using the client.

```bash
psql $ENDPOINT-FROM-SECRET-BASE64-DECODED
```

Insert some data into the database.

```sql
create table t_random as select s, md5(random()::text) from generate_Series(1,5) s;
```

Adjust the backup to target the GUID of the PostgreSQL instance.

Create a backup.

```bash
watch kubectl apply -f examples/sample/backup-example.yaml  -o yaml
```

Add some more data. We would expect this data to not be in the database after the restore.

```sql
create table nothing_here as select s, md5(random()::text) from generate_Series(1,5) s;
```

Restore the database to the previous state.

```bash
watch kubectl apply -f examples/sample/restore-example.yaml  -o yaml
```

Check that the data has been reverted to the state at the time of backup.
