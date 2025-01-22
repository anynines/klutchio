#!/bin/bash

set -euo pipefail

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <instance-name>"
    echo ""
    echo "Example:"
    echo "  $0 postgresql-ms-123456789"
    echo ""
    echo "This script starts an SSH tunnel for the given a9s data service instance."
    echo "To get a <instance-name>, you can create one via the #create-data-service slack channel."
    echo ""
    echo "The local ports will be chosen based on the type of instance, to be compatible with the"
    echo "example configurations in provider-anynines/examples/provider/."
    exit 127
fi

instance_name="${1}"

instance_type=$(echo "$instance_name" | cut -d '-' -f 1)

ports=$(
    cat <<EOF | grep "^${instance_type}" || ( echo "Unknown instance type: ${instance_type}" ; false )
postgresql	8989	8988
search		9189	9188
mongodb		9389	9388
logme2		9489	9488
mariadb		9589	9588
rabbitmq	9689	9688 # aka messaging
prometheus     	9789	9788
EOF
)

broker_port=$(echo "$ports" | awk '{print $2}')
backup_port=$(echo "$ports" | awk '{print $3}')

echo "Retrieving IPs..."

instances=$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d ${instance_name} instances")

broker_service_ip=$(echo "$instances" | grep broker/ | awk '{print $4}')
echo "Got broker IP: ${broker_service_ip}"
backup_service_ip=$(echo "$instances" | grep backup-manager/ | awk '{print $4}')
echo "Got backup IP: ${backup_service_ip}"

echo ""
echo "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~"
echo "  Exposing ${instance_name} on local ports ${broker_port} and ${backup_port}"
echo "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~"
echo ""
ssh -L "${broker_port}:${broker_service_ip}:3000" \
    -L "${backup_port}:${backup_service_ip}:3000" \
    -o "ServerAliveInterval 30" \
    -o "ServerAliveCountMax 3" \
    aws-s1-inception