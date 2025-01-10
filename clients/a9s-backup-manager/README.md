# A9s Backup Manager

## Pre-req

* Follow [this](https://anynines.atlassian.net/wiki/spaces/A8S/pages/3067969555/a8s+Introduction+to+create-data-service+Channel#How-to-get-access) link to get access to `aws-inception`.

## Testing the client

[//]: # (TODO: Add more details or link to provider-anynines for more details.)

* Create a data service by executing the `/create postgresql` command in
 [create-data-service slack channel].
* After which create a postgres service instance using the crossplane Postgresql managed resource
or by any other method.
* Once service broker has been successfully created run the following cmds in
a terminal to expose the backup-manager-api port locally on your system
(*do not close the terminal else the connection would be closed*).
  * Set the values of {service-instance-name}(the name of the service instance you want to connect
    to e.g. postgresql-ms-1234567890) and {localport}(the local port you want to expose e.g. 8989).

``` bash
# Set service instance name e.g. postgresql-ms-1234567890
export SERVICE_INSTANCE_NAME={service-instance-name} 

# Get IP of the service backup manager
export BACKUP_MANAGER_IP=$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $SERVICE_INSTANCE_NAME instances | grep backup-manager" | awk '{print $4}')

# Create ssh tunnel to pg servicebroker
ssh -L {localport}:$BACKUP_MANAGER_IP:3000 aws-s1-inception -o "ServerAliveInterval 30" -o "ServerAliveCountMax 3"
```

* You can locally import the username and password required to communicate with the
 api using the following cmds.

``` bash
# Set service instance name e.g. postgresql-ms-1234567890
export SERVICE_INSTANCE_NAME={service-instance-name} 

# Get creds for backup manager
export BACKUP_MANAGER_USERNAME=$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r')

export BACKUP_MANAGER_PASSWORD=$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r')

echo bkp-mgr-username=$BACKUP_MANAGER_USERNAME
echo bkp-mgr-password=$BACKUP_MANAGER_PASSWORD
```

* Now you can create a `main.go` file to test out the implemented functionality e.g.

``` go
package main

import (
 "fmt"

 backupmanager "github.com/anynines/klutchio/clients/a9s-backup-manager"
)

func main() {
 config := &backupmanager.ClientConfiguration{
  URL: "http://localhost:8989",
  AuthConfig: &backupmanager.AuthConfig{
   BasicAuthConfig: &backupmanager.BasicAuthConfig{
    Username: "username",
    Password: "password",
   },
  },
  TimeoutSeconds: 10,
  Verbose:        true,
 }
 client, err := backupmanager.NewClient(config)
 if err != nil {
  panic(err)
 }

 req := backupmanager.CreateBackupRequest{
  InstanceID: "<Insert instanceID here e.g. asdasd34-5d89-8564-7af9-isg87fko85k67>",
 }

 resp, err := client.CreateBackup(&req)
 if err != nil {
  panic(err)
 }

 fmt.Println(resp)
}
```

[create-data-service slack channel]: https://anynines-gmbh.slack.com/archives/C02EJBT3947
