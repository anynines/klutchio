
default:
.PHONY: default

cleanup-clusters:
	@echo "Will delete the following clusters:"
	@kind get clusters | grep '^cb-e2e-' || ( echo " (none found)"; false )
	@echo "Press ENTER to confirm, CTRL+C to abort"
	@read
	@for cluster in $(shell kind get clusters | grep '^cb-e2e-'); do kind delete cluster --name=$$cluster ; done

.PHONY: cleanup-clusters
