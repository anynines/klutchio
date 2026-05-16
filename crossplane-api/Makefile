IMAGE_NAME := dataservices
ECR_REPO := public.ecr.aws/w5n9a2g2/klutch

.PHONY: providerconfig
providerconfig:
	@if [ -z "$(postgresInstanceName)$(keyvalueInstanceName)$(mongodbInstanceName)$(searchInstanceName)$(mariadbInstanceName)$(messagingInstanceName)$(logme2InstanceName)$(prometheusInstanceName)" ]; then \
		echo "ERROR: Please provide at least one instance name variable (e.g., postgresInstanceName, mongodbInstanceName, etc.)"; \
		exit 1; \
	fi
# Populate Secrets and create PostgreSQL ProvideConfig using the instance name
ifdef postgresInstanceName
	@export PG_SERVICE_INSTANCE_NAME=$$postgresInstanceName; \
	if [[ $$GET_BROKER_IP == "true" ]]; then \
		if [[ $$PG_SERVICE_INSTANCE_NAME == "" ]]; then \
			echo "ERROR: postgresInstanceName must not be empty!"; \
			exit 1; \
		fi; \
		export PG_SERVICEBROKER_IP=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$PG_SERVICE_INSTANCE_NAME instances" --json | yq -p=json '.Tables.0.Rows.[] | select ( .instance == "broker/*") | .ips'); \
		export PG_SERVICEBROKER_HOST="http://$$PG_SERVICEBROKER_IP:3000"; \
		export PG_BACKUP_MANAGER_IP=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$PG_SERVICE_INSTANCE_NAME instances" --json | yq -p=json '.Tables.0.Rows.[] | select ( .instance == "backup-manager/*") | .ips'); \
		export PG_BACKUP_MANAGER_HOST="http://$$PG_BACKUP_MANAGER_IP:3000"; \
		echo "Backup Manager IP: $$PG_BACKUP_MANAGER_IP"; \
		echo "Service Broker IP: $$PG_SERVICEBROKER_IP"; \
		echo "Backup Manager Host: $$PG_BACKUP_MANAGER_HOST"; \
		echo "Service Broker Host: $$PG_SERVICEBROKER_HOST"; \
		if [ -z $$PG_BACKUP_MANAGER_IP ]; then \
			COMPONENT_IP_NOT_RETRIEVABLE="true"; \
			echo "ERROR: IP adress could not be retrieved for PostgreSQL Backup Manager"; \
		fi; \
		if [ -z $$PG_SERVICEBROKER_IP ]; then \
			COMPONENT_IP_NOT_RETRIEVABLE="true"; \
			echo "ERROR: IP adress could not be retrieved for PostgreSQL Service Broker"; \
		fi; \
		if [ ! -z $$COMPONENT_IP_NOT_RETRIEVABLE ]; then \
			exit 1; \
		fi; \
	else \
		export PG_BACKUP_MANAGER_HOST="http://dockerhost:8988"; \
		export PG_SERVICEBROKER_HOST="http://dockerhost:8989"; \
	fi; \
	export PG_SERVICEBROKER_USERNAME=admin; \
	export PG_SERVICEBROKER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;credhub get -n /postgresql_service_broker_password | grep value:" | awk '{print $2}'); \
	export PG_BACKUP_MANAGER_USERNAME==$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$PG_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r'); \
	export PG_BACKUP_MANAGER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$PG_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r'); \
	export BASE64ENCODED_POSTGRESQL_SERVICE_BROKER_USERNAME=$$(echo "$$PG_SERVICEBROKER_USERNAME" | base64); \
	export BASE64ENCODED_POSTGRESQL_SERVICE_BROKER_PASSWORD=$$(echo "$$PG_SERVICEBROKER_PASSWORD" | sed 's/^value: //' | base64); \
	export BASE64ENCODED_POSTGRESQL_BACKUP_MANAGER_USERNAME=$$(echo "$$PG_BACKUP_MANAGER_USERNAME" | sed 's/^=//' | base64); \
	export BASE64ENCODED_POSTGRESQL_BACKUP_MANAGER_PASSWORD=$$(echo "$$PG_BACKUP_MANAGER_PASSWORD" | base64); \
	envsubst < ../provider-anynines/examples/provider/config-postgres.yaml | kubectl apply -f - && \
	echo "Successfully created ProviderConfig for postgres instance: $$postgresInstanceName" || \
	echo "There was a problem creating the ProviderConfig for postgres instance: $$postgresInstanceName"
endif
# Populate Secrets and create KeyValue ProviderConfig using the instance name
ifdef keyvalueInstanceName
	@export KEY_VALUE_SERVICE_INSTANCE_NAME=$$keyvalueInstanceName; \
	if [[ $$GET_BROKER_IP == "true" ]]; then \
		if [[ $$KEY_VALUE_SERVICE_INSTANCE_NAME == "" ]]; then \
			echo "ERROR: keyvalueInstanceName must not be empty!"; \
			exit 1; \
		fi; \
		export KV_SERVICEBROKER_IP=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$KEY_VALUE_SERVICE_INSTANCE_NAME instances" --json | yq -p=json '.Tables.0.Rows.[] | select ( .instance == "broker/*") | .ips'); \
		export KV_SERVICEBROKER_HOST="http://$$KV_SERVICEBROKER_IP:3000"; \
		export KV_BACKUP_MANAGER_IP=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$KEY_VALUE_SERVICE_INSTANCE_NAME instances" --json | yq -p=json '.Tables.0.Rows.[] | select ( .instance == "backup-manager/*") | .ips'); \
		export KV_BACKUP_MANAGER_HOST="http://$$KV_BACKUP_MANAGER_IP:3000"; \
		echo "Backup Manager IP: $$KV_BACKUP_MANAGER_IP"; \
		echo "Service Broker IP: $$KV_SERVICEBROKER_IP"; \
		echo "Backup Manager Host: $$KV_BACKUP_MANAGER_HOST"; \
		echo "Service Broker Host: $$KV_SERVICEBROKER_HOST"; \
		if [ -z $$KV_BACKUP_MANAGER_IP ]; then \
			COMPONENT_IP_NOT_RETRIEVABLE="true"; \
			echo "ERROR: IP adress could not be retrieved for KeyValue Backup Manager"; \
		fi; \
		if [ -z $$KV_SERVICEBROKER_IP ]; then \
			COMPONENT_IP_NOT_RETRIEVABLE="true"; \
			echo "ERROR: IP adress could not be retrieved for KeyValue Service Broker"; \
		fi; \
		if [ ! -z $$COMPONENT_IP_NOT_RETRIEVABLE ]; then \
			exit 1; \
		fi; \
	else \
		export KV_BACKUP_MANAGER_HOST="http://dockerhost:8988"; \
		export KV_SERVICEBROKER_HOST="http://dockerhost:8989"; \
	fi; \
	export KEYVALUE_SERVICEBROKER_USERNAME=admin; \
	export KEYVALUE_SERVICEBROKER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;credhub get -n /keyvalue_service_broker_password | grep value:" | awk '{print $2}'); \
	export KEYVALUE_SERVICE_INSTANCE_NAME=$$keyvalueInstanceName; \
	export KEYVALUE_BACKUP_MANAGER_USERNAME=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$KEYVALUE_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r'); \
	export KEYVALUE_BACKUP_MANAGER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$KEYVALUE_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r'); \
	export BASE64ENCODED_KEYVALUE_SERVICE_BROKER_USERNAME=$$(echo "$$KEYVALUE_SERVICEBROKER_USERNAME" | base64); \
	export BASE64ENCODED_KEYVALUE_SERVICE_BROKER_PASSWORD=$$(echo "$$KEYVALUE_SERVICEBROKER_PASSWORD" | sed 's/^value: //' | base64); \
	export BASE64ENCODED_KEYVALUE_BACKUP_MANAGER_USERNAME=$$(echo "$$KEYVALUE_BACKUP_MANAGER_USERNAME" | base64); \
	export BASE64ENCODED_KEYVALUE_BACKUP_MANAGER_PASSWORD=$$(echo "$$KEYVALUE_BACKUP_MANAGER_PASSWORD" | base64); \
	envsubst < ../provider-anynines/examples/provider/config-keyvalue.yaml | kubectl apply -f - ; \
	echo "Successfully created ProviderConfig for keyvalue instance: $$keyvalueInstanceName";
endif
# Populate Secrets and create Mongodb ProviderConfig using the instance name
ifdef mongodbInstanceName
	@export MONGODB_SERVICEBROKER_USERNAME=admin; \
	export MONGODB_SERVICEBROKER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;credhub get -n /mongodb_service_broker_password | grep value:" | awk '{print $2}'); \
	export MONGODB_SERVICE_INSTANCE_NAME=$$mongodbInstanceName; \
	export MONGODB_BACKUP_MANAGER_USERNAME=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$MONGODB_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r'); \
	export MONGODB_BACKUP_MANAGER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$MONGODB_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r'); \
	export BASE64ENCODED_MONGODB_SERVICE_BROKER_USERNAME=$$(echo "$$MONGODB_SERVICEBROKER_USERNAME" | base64); \
	export BASE64ENCODED_MONGODB_SERVICE_BROKER_PASSWORD=$$(echo "$$MONGODB_SERVICEBROKER_PASSWORD" | sed 's/^value: //' | base64); \
	export BASE64ENCODED_MONGODB_BACKUP_MANAGER_USERNAME=$$(echo "$$MONGODB_BACKUP_MANAGER_USERNAME" | base64); \
	export BASE64ENCODED_MONGODB_BACKUP_MANAGER_PASSWORD=$$(echo "$$MONGODB_BACKUP_MANAGER_PASSWORD" | base64); \
	envsubst < ../provider-anynines/examples/provider/config-mongodb.yaml | kubectl apply -f - ; \
	echo "Successfully created ProviderConfig for mongodb instance: $$mongodbInstanceName";
endif
# Populate Secrets and create Search ProviderConfig using the instance name
ifdef searchInstanceName
	@export SEARCH_SERVICEBROKER_USERNAME=admin; \
	export SEARCH_SERVICEBROKER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;credhub get -n /search_service_broker_password | grep value:" | awk '{print $2}'); \
	export SEARCH_SERVICE_INSTANCE_NAME=$$searchInstanceName; \
	export SEARCH_BACKUP_MANAGER_USERNAME==$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$SEARCH_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r'); \
	export SEARCH_BACKUP_MANAGER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$SEARCH_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r'); \
	export BASE64ENCODED_SEARCH_SERVICE_BROKER_USERNAME=$$(echo "$$SEARCH_SERVICEBROKER_USERNAME" | base64); \
	export BASE64ENCODED_SEARCH_SERVICE_BROKER_PASSWORD=$$(echo "$$SEARCH_SERVICEBROKER_PASSWORD" | sed 's/^value: //' | base64); \
	export BASE64ENCODED_SEARCH_BACKUP_MANAGER_USERNAME=$$(echo "$$SEARCH_BACKUP_MANAGER_USERNAME" | sed 's/^=//' | base64); \
	export BASE64ENCODED_SEARCH_BACKUP_MANAGER_PASSWORD=$$(echo "$$SEARCH_BACKUP_MANAGER_PASSWORD" | base64); \
	envsubst < ../provider-anynines/examples/provider/config-search.yaml | kubectl apply -f - ; \
	echo "Successfully created ProviderConfig for search instance: $$searchInstanceName";
endif
# Populate Secrets and create Mariadb ProviderConfig using the instance name
ifdef mariadbInstanceName
	@export MARIADB_SERVICEBROKER_USERNAME=admin; \
	export MARIADB_SERVICEBROKER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;credhub get -n /mariadb_service_broker_password | grep value:" | awk '{print $2}'); \
	export MARIADB_SERVICE_INSTANCE_NAME=$$mariadbInstanceName; \
	export MARIADB_BACKUP_MANAGER_USERNAME=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$MARIADB_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r'); \
	export MARIADB_BACKUP_MANAGER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$MARIADB_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r'); \
	export BASE64ENCODED_MARIADB_SERVICE_BROKER_USERNAME=$$(echo "$$MARIADB_SERVICEBROKER_USERNAME" | base64); \
	export BASE64ENCODED_MARIADB_SERVICE_BROKER_PASSWORD=$$(echo "$$MARIADB_SERVICEBROKER_PASSWORD" | sed 's/^value: //' | base64); \
	export BASE64ENCODED_MARIADB_BACKUP_MANAGER_USERNAME=$$(echo "$$MARIADB_BACKUP_MANAGER_USERNAME" | base64); \
	export BASE64ENCODED_MARIADB_BACKUP_MANAGER_PASSWORD=$$(echo "$$MARIADB_BACKUP_MANAGER_PASSWORD" | base64); \
	envsubst < ../provider-anynines/examples/provider/config-mariadb.yaml | kubectl apply -f - ; \
	echo "Successfully created ProviderConfig for mariadb instance: $$mariadbInstanceName";
endif
# Populate Secrets and create Messaging ProviderConfig using the instance name
ifdef messagingInstanceName
	@export MESSAGING_SERVICEBROKER_USERNAME=admin; \
	export MESSAGING_SERVICEBROKER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;credhub get -n /rabbitmq_service_broker_password | grep value:" | awk '{print $2}'); \
	export MESSAGING_SERVICE_INSTANCE_NAME=$$messagingInstanceName; \
	export MESSAGING_BACKUP_MANAGER_USERNAME=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$MESSAGING_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r'); \
	export MESSAGING_BACKUP_MANAGER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$MESSAGING_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r'); \
	export BASE64ENCODED_MESSAGING_SERVICE_BROKER_USERNAME=$$(echo "$$MESSAGING_SERVICEBROKER_USERNAME" | base64); \
	export BASE64ENCODED_MESSAGING_SERVICE_BROKER_PASSWORD=$$(echo "$$MESSAGING_SERVICEBROKER_PASSWORD" | sed 's/^value: //' | base64); \
	export BASE64ENCODED_MESSAGING_BACKUP_MANAGER_USERNAME=$$(echo "$$MESSAGING_BACKUP_MANAGER_USERNAME" | base64); \
	export BASE64ENCODED_MESSAGING_BACKUP_MANAGER_PASSWORD=$$(echo "$$MESSAGING_BACKUP_MANAGER_PASSWORD" | base64); \
	envsubst < ../provider-anynines/examples/provider/config-messaging.yaml | kubectl apply -f - ; \
	echo "Successfully created ProviderConfig for messaging instance: $$messagingInstanceName";
endif
# Populate Secrets and create Logme2 ProviderConfig using the instance name
ifdef logme2InstanceName
	@export LOGME2_SERVICEBROKER_USERNAME=admin; \
	export LOGME2_SERVICEBROKER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;credhub get -n /logme2_service_broker_password | grep value:" | awk '{print $2}'); \
	export LOGME2_SERVICE_INSTANCE_NAME=$$logme2InstanceName; \
	export LOGME2_BACKUP_MANAGER_USERNAME=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$LOGME2_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r'); \
	export LOGME2_BACKUP_MANAGER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$LOGME2_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r'); \
	export BASE64ENCODED_LOGME2_SERVICE_BROKER_USERNAME=$$(echo "$$LOGME2_SERVICEBROKER_USERNAME" | base64); \
	export BASE64ENCODED_LOGME2_SERVICE_BROKER_PASSWORD=$$(echo "$$LOGME2_SERVICEBROKER_PASSWORD" | sed 's/^value: //' | base64); \
	export BASE64ENCODED_LOGME2_BACKUP_MANAGER_USERNAME=$$(echo "$$LOGME2_BACKUP_MANAGER_USERNAME" | base64); \
	export BASE64ENCODED_LOGME2_BACKUP_MANAGER_PASSWORD=$$(echo "$$LOGME2_BACKUP_MANAGER_PASSWORD" | base64); \
	envsubst < ../provider-anynines/examples/provider/config-logme2.yaml | kubectl apply -f - ; \
	echo "Successfully created ProviderConfig for logme2 instance: $$logme2InstanceName";
endif
# Populate Secrets and create Prometheus ProviderConfig using the instance name
ifdef prometheusInstanceName
	@export PROMETHEUS_SERVICEBROKER_USERNAME=admin; \
	export PROMETHEUS_SERVICEBROKER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;credhub get -n /prometheus_service_broker_password | grep value:" | awk '{print $2}'); \
	export PROMETHEUS_SERVICE_INSTANCE_NAME=$$prometheusInstanceName; \
	export PROMETHEUS_BACKUP_MANAGER_USERNAME=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$PROMETHEUS_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_USERNAME= | cut -d '=' -f2 | tr -d '\r'); \
	export PROMETHEUS_BACKUP_MANAGER_PASSWORD=$$(ssh aws-s1-inception ". /var/vcap/store/jumpbox/home/a9s/bosh/envs/dsf2;bosh -d $$PROMETHEUS_SERVICE_INSTANCE_NAME ssh backup-manager -c 'cat /var/vcap/jobs/anynines-backup-manager/config/export_env.sh'" | grep HTTP_PASSWORD= | cut -d '=' -f2 | tr -d '\r'); \
	export BASE64ENCODED_PROMETHEUS_SERVICE_BROKER_USERNAME=$$(echo "$$PROMETHEUS_SERVICEBROKER_USERNAME" | base64); \
	export BASE64ENCODED_PROMETHEUS_SERVICE_BROKER_PASSWORD=$$(echo "$$PROMETHEUS_SERVICEBROKER_PASSWORD" | sed 's/^value: //' | base64); \
	export BASE64ENCODED_PROMETHEUS_BACKUP_MANAGER_USERNAME=$$(echo "$$PROMETHEUS_BACKUP_MANAGER_USERNAME" | base64); \
	export BASE64ENCODED_PROMETHEUS_BACKUP_MANAGER_PASSWORD=$$(echo "$$PROMETHEUS_BACKUP_MANAGER_PASSWORD" | base64); \
	envsubst < ../provider-anynines/examples/provider/config-prometheus.yaml | kubectl apply -f - ; \
	echo "Successfully created ProviderConfig for prometheus instance: $$prometheusInstanceName";
endif

dataservices-config-push:
ifdef dataservicesConfigVersion
		# Build and push configuration package
		crossplane xpkg build -f ./api
		crossplane xpkg push -f ./api/anynines-dataservices-*.xpkg $(ECR_REPO)/$(IMAGE_NAME):$(dataservicesConfigVersion)
		rm ./api/anynines-dataservices-*
endif
.PHONY: providerconfig

.PHONY: check-version-set
check-version-set:
ifndef VERSION
	$(error VERSION is not set. Please set it to the version you are releasing, e.g. make release VERSION=v1.0.0)
endif
	@if ! echo "$(VERSION)" | grep -q -E '^v[0-9]+\.[0-9]+\.[0-9]'; then \
		echo "Error! The value of VERSION must follow SemVer rules."; \
		exit 1; \
	fi


.PHONY: release
release: check-version-set require-crossplane ## Build and push data services config and provider-anynines images
ifndef AWS_PROFILE_NAME
	@echo "AWS_PROFILE_NAME is not set, using active AWS profile $$(aws configure list | grep "profile" | cut -d ':' -f 2 | awk '{$$1=$$1};1')"
	@echo "You can set the AWS_PROFILE_NAME make variable to use a different profile, e.g. make release VERSION=$(VERSION) AWS_PROFILE_NAME=ECR"
endif
	@aws ecr-public get-login-password --region us-east-1 $(if $(AWS_PROFILE_NAME),--profile=$(AWS_PROFILE_NAME)) | docker login --username AWS --password-stdin public.ecr.aws/w5n9a2g2
	@$(MAKE) dataservices-config-push dataservicesConfigVersion=$(VERSION)
	@$(MAKE) -C ../provider-anynines provider-controller-push IMAGETAG=$(VERSION)
	@$(MAKE) -C ../provider-anynines provider-build-push IMAGETAG=$(VERSION)
	@echo "Updating version references in deploy/ and README.md..."

.PHONY: update-version-refs
update-version-refs: check-version-set
# update a9s DS Configuration manifest
	@sed -E 's|klutch/dataservices:v[0-9]+\.[0-9]+\.[0-9]+[^"]*|klutch/dataservices:$(VERSION)|g' deploy/config-pkg-anynines.yaml > deploy/config-pkg-anynines.yaml.tmp && mv deploy/config-pkg-anynines.yaml.tmp deploy/config-pkg-anynines.yaml
# update provider-anynines Provider manifest
	@sed -E 's|klutch/provider-anynines:v[0-9]+\.[0-9]+\.[0-9]+[^"]*|klutch/provider-anynines:$(VERSION)|g' deploy/provider-anynines.yaml > deploy/provider-anynines.yaml.tmp && mv deploy/provider-anynines.yaml.tmp deploy/provider-anynines.yaml
# update Readme installation instruction
	@sed -E 's|klutch/dataservices:v[0-9]+\.[0-9]+\.[0-9]+[^"]*|klutch/dataservices:$(VERSION)|g' README.md > README.md.tmp && mv README.md.tmp README.md
	@echo "Version references updated to $(VERSION)."

.PHONY: require-%
require-%:
	@if ! command -v $* 1> /dev/null 2>&1; then echo "$* not found in \$$PATH"; exit 1; fi

