# ====================================================================================
# Setup Project
PROJECT_NAME := provider-anynines
PROJECT_REPO := github.com/anynines/klutch/$(PROJECT_NAME)
ECR_ACCOUNT := public.ecr.aws/w5n9a2g2
ECR_FOLDER_PATH := klutch
ECR_REPO := $(ECR_ACCOUNT)/$(ECR_FOLDER_PATH)
BUILDX_NAME := provider-builder

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis pkg
GO111MODULE = on
-include build/makelib/golang.mk

# kind-related versions
KIND_VERSION ?= v0.12.0
KIND_NODE_IMAGE_TAG ?= v1.23.4

# ====================================================================================
# Setup Kubernetes tools

-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Images

DOCKER_REGISTRY ?= crossplane
IMAGES = $(PROJECT_NAME) 
-include build/makelib/imagelight.mk

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# integration tests
e2e.run: test-integration

# NOTE(hasheddan): we must ensure up is installed in tool cache prior to build
# as including the k8s_tools machinery prior to the xpkg machinery sets UP to
# point to tool cache.
build.init: $(UP)

# Run integration tests.
test-integration: $(KIND) $(KUBECTL) $(UP) $(HELM3)
	@$(INFO) running integration tests using kind $(KIND_VERSION)
	@KIND_NODE_IMAGE_TAG=${KIND_NODE_IMAGE_TAG} $(ROOT_DIR)/cluster/local/integration_tests.sh || $(FAIL)
	@$(OK) integration tests passed

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# NOTE(hasheddan): the build submodule currently overrides XDG_CACHE_HOME in
# order to force the Helm 3 to use the .work/helm directory. This causes Go on
# Linux machines to use that directory as the build cache as well. We should
# adjust this behavior in the build submodule because it is also causing Linux
# users to duplicate their build cache, but for now we just make it easier to
# identify its location in CI so that we cache between builds.
go.cachedir:
	@go env GOCACHE

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	kubectl apply -f deploy/provider-cluster-role.yaml
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/provider --debug

dev: $(KIND) $(KUBECTL)
	@$(INFO) Creating kind cluster
	@$(KIND) create cluster --name=$(PROJECT_NAME)-dev
	@$(KUBECTL) cluster-info --context kind-$(PROJECT_NAME)-dev
	@$(INFO) Installing Crossplane CRDs
	@$(KUBECTL) apply -k https://github.com/crossplane/crossplane//cluster?ref=master
	@$(INFO) Installing Provider anynines CRDs
	@$(KUBECTL) apply -R -f package/crds
	@$(INFO) Starting Provider anynines controllers
	@$(GO) run cmd/provider/main.go --debug

dev-clean: $(KIND) $(KUBECTL)
	@$(INFO) Deleting kind cluster
	@$(KIND) delete cluster --name=$(PROJECT_NAME)-dev

.PHONY: submodules fallthrough test-integration run dev dev-clean

# ====================================================================================
# Special Targets

# Install gomplate
GOMPLATE_VERSION := 3.10.0
GOMPLATE := $(TOOLS_HOST_DIR)/gomplate-$(GOMPLATE_VERSION)

$(GOMPLATE):
	@$(INFO) installing gomplate $(SAFEHOSTPLATFORM)
	@mkdir -p $(TOOLS_HOST_DIR)
	@curl -fsSLo $(GOMPLATE) https://github.com/hairyhenderson/gomplate/releases/download/v$(GOMPLATE_VERSION)/gomplate_$(SAFEHOSTPLATFORM) || $(FAIL)
	@chmod +x $(GOMPLATE)
	@$(OK) installing gomplate $(SAFEHOSTPLATFORM)

export GOMPLATE

# This target prepares repo for your provider by replacing all "anynines"
# occurrences with your provider name.
# This target can only be run once, if you want to rerun for some reason,
# consider stashing/resetting your git state.
# Arguments:
#   provider: Camel case name of your provider, e.g. GitHub, PlanetScale
provider.prepare:
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@PROVIDER=$(provider) ./hack/helpers/prepare.sh

# This target adds a new api type and its controller.
# You would still need to register new api in "apis/<provider>.go" and
# controller in "internal/controller/<provider>.go".
# Arguments:
#   provider: Camel case name of your provider, e.g. GitHub, PlanetScale
#   group: API group for the type you want to add.
#   kind: Kind of the type you want to add
#	apiversion: API version of the type you want to add. Optional and defaults to "v1alpha1"
provider.addtype: $(GOMPLATE)
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@[ "${group}" ] || ( echo "argument \"group\" is not set"; exit 1 )
	@[ "${kind}" ] || ( echo "argument \"kind\" is not set"; exit 1 )

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run crossplane locally, out-of-cluster. Useful for development.

endef
# The reason CROSSPLANE_MAKE_HELP is used instead of CROSSPLANE_HELP is because the crossplane
# binary will try to use CROSSPLANE_HELP if it is set, and this is for something different.
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special

# Upload the provider controllers' images to ECR for both amd & arm architectures.
provider-controller-push:
	@if [ -z "$(IMAGETAG)" ]; then \
		$(INFO) "ERROR: IMAGETAG is not set."; \
		exit 1; \
	fi
	@for arch in $(IMAGE_ARCHS); do \
		docker tag $(BUILD_REGISTRY)/$(PROJECT_NAME)-$$arch:latest $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG)-$$arch; \
		docker push $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG)-$$arch; \
		$(INFO) Image: $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG)-$$arch pushed to ECR; \
	done

# There also exists a `docker manifest create` command that does not require a buildx container but since it's an
# experimental feature I opted to use the more tried and tested `docker buildx create`
	$(INFO) "Build flags set to $(BUILDX_CREATE_FLAGS)"
	-@docker buildx create \
    	--name=$(BUILDX_NAME) \
    	$(BUILDX_CREATE_FLAGS)
	@if [ $$? ] ; then \
		echo "Builder already $(BUILDX_NAME) existed."; \
	else \
		echo "Builder $(BUILDX_NAME) newly created"; \
		touch builder_created.flag; \
	fi


	docker buildx imagetools create -t $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG) $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG)-$(word 1, $(IMAGE_ARCHS)) $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG)-$(word 2, $(IMAGE_ARCHS))
	$(INFO) Image: $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG) manifest list pushed to ECR;

	@if [ -f builder_created.flag ]; then \
		docker buildx rm $(BUILDX_NAME) > /dev/null; \
		echo "removed Builder $(BUILDX_NAME) that was created by this make command"; \
	else \
		echo "Builder $(BUILDX_NAME) was already there, leaving it as is."; \
	fi
	@rm -f builder_created.flag


# Upload a multi-arch provider controllers' images to ECR for both amd & arm architectures.
provider-controller-build-push:
	$(INFO) "Build flags: $(BUILDX_CREATE_FLAGS) Image Arch: $(IMAGE_PLATFORMS)"
	-@docker buildx create \
    	--name=$(BUILDX_NAME) \
    	$(BUILDX_CREATE_FLAGS)
	@if [ $$? ] ; then \
		echo "Builder already $(BUILDX_NAME) existed."; \
	else \
		echo "Builder $(BUILDX_NAME) newly created"; \
		touch builder_created.flag; \
	fi

	@cp ./cluster/images/provider-anynines/Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp $(OUTPUT_DIR)/bin/$(OS)_$(ARCH)/provider $(IMAGE_TEMP_DIR) || $(FAIL)
	@cd $(IMAGE_TEMP_DIR) && $(SED_CMD) 's|BASEIMAGE|$(OSBASEIMAGE)|g' Dockerfile || $(FAIL)
	docker buildx build -o type=image --builder=$(BUILDER_NAME) \
        --platform=$(IMAGE_PLATFORMS) \
        --push -t $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG) \
        $(IMAGE_TEMP_DIR) || $(FAIL)

	$(INFO) Image: $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG) manifest list pushed to ECR;

	@if [ -f builder_created.flag ]; then \
		docker buildx rm $(BUILDX_NAME) > /dev/null; \
		echo "removed Builder $(BUILDX_NAME) that was created by this make command"; \
	else \
		echo "Builder $(BUILDX_NAME) was already there, leaving it as is."; \
	fi
	@rm -f builder_created.flag

# Build a single Provider configuration package for both amd & arm architectures and upload it to AWS.
provider-build-push:
	@if [ -z "$(IMAGETAG)" ]; then \
		$(INFO) "ERROR: IMAGETAG is not set."; \
		exit 1; \
	fi
	cp package/crossplane.yaml bu.yaml
	cat package/crossplane.yaml | sed "s/\VERSION/$(IMAGETAG)/g" > package/temp_crossplane.yaml
	mv package/temp_crossplane.yaml package/crossplane.yaml
	crossplane xpkg build --package-file package/$(PROJECT_NAME).xpkg --package-root package
	$(INFO) "Building provider image"
	crossplane xpkg push --package-files package/$(PROJECT_NAME).xpkg $(ECR_REPO)/$(PROJECT_NAME):$(IMAGETAG)
	$(INFO) "image $(PROJECT_NAME):$(IMAGETAG) pushed to ECR"
	$(INFO) "removing local package file"
	rm package/$(PROJECT_NAME).xpkg
	mv bu.yaml package/crossplane.yaml
