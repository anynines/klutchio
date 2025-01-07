# ====================================================================================
# Setup Project

PROJECT_NAME := provider-anynines
ECR_ACCOUNT := public.ecr.aws/w5n9a2g2
ECR_FOLDER_PATH := klutch
ECR_REPO := $(ECR_ACCOUNT)/$(ECR_FOLDER_PATH)
PLATFORMS := linux_amd64 linux_arm64

include ../../../build/makelib/common.mk
include ../../../build/makelib/imagelight.mk
include ../../../ci/common.mk
include ../../../ci/imagelight.mk

# ====================================================================================
#  Options
DOCKER_REGISTRY ?= crossplane
IMAGE = $(BUILD_REGISTRY)/provider-anynines-$(ARCH)
OSBASEIMAGE = scratch
BUILDER_NAME ?= provider-anynines-buildx-builder

# ====================================================================================
# Targets

img.build:
	@$(INFO) docker build $(IMAGE)
	@$(MAKE) BUILD_ARGS="--load" img.build.shared
	@$(OK) docker build $(IMAGE)

img.publish:
	@$(INFO) Skipping image publish for $(IMAGE)
	@echo Publish is deferred to xpkg machinery
	@$(OK) Image publish skipped for $(IMAGE)

img.build.shared:
	@cp Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -r $(OUTPUT_DIR)/bin/ $(IMAGE_TEMP_DIR)/bin || $(FAIL)
	@docker buildx build $(BUILD_ARGS) \
		--platform $(IMAGE_PLATFORMS) \
		-t $(IMAGE) \
		$(IMAGE_TEMP_DIR) || $(FAIL)

img.promote:
	@$(INFO) Skipping image promotion from $(FROM_IMAGE) to $(TO_IMAGE)
	@echo Promote is deferred to xpkg machinery
	@$(OK) Image promotion skipped for $(FROM_IMAGE) to $(TO_IMAGE)

img.buildx:
	@$(INFO) docker buildx build $(IMAGE) $(BUILDX_CREATE_FLAGS)
	@cp Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -R $(ROOT_DIR)/package $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -R $(OUTPUT_DIR)/bin $(IMAGE_TEMP_DIR) || $(FAIL)
	@cd $(IMAGE_TEMP_DIR) && $(SED_CMD) 's|BASEIMAGE|$(OSBASEIMAGE)|g' Dockerfile || $(FAIL)
	@cd $(IMAGE_TEMP_DIR) && $(SED_CMD) 's|VERSION|$(VERSION)|g' package/crossplane.yaml || $(FAIL)
	@cd $(IMAGE_TEMP_DIR) && $(SED_CMD) 's|DOCKER_REGISTRY|$(DOCKER_REGISTRY)|g' package/crossplane.yaml || $(FAIL)
	@cd $(IMAGE_TEMP_DIR) && find package -type f -name '*.yaml' -exec cat {} >> 'package.yaml' \; -exec printf '\n---\n' \; || $(FAIL)
	@docker buildx create \
    	--name=$(BUILDER_NAME) \
    	$(BUILDX_CREATE_FLAGS) || echo "Builder $(BUILDER_NAME) already exists"
	docker buildx build -o type=image --builder=$(BUILDER_NAME) \
        --platform=linux/$(ARCH) \
        --push -t $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG)-$(ARCH) \
        $(IMAGE_TEMP_DIR) || $(FAIL)
	@$(OK) docker buildx build $(IMAGE)
