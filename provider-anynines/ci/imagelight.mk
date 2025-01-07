# Options

# ====================================================================================
# Common Targets

do.buildx.image.%:
	@$(MAKE) -C $(IMAGE_DIR)/$* IMAGE_PLATFORMS=$(IMAGE_PLATFORM) IMAGE=$(BUILD_REGISTRY)/$*-$(ARCH) img.buildx
do.buildx.images: $(foreach i,$(IMAGES), do.buildx.image.$(i))
do.skipx.images:
	@$(OK) Skipping image build for unsupported platform $(IMAGE_PLATFORM)

buildx.merge:
	@docker buildx create \
    	--name=$(BUILDER_NAME) \
    	$(BUILDX_CREATE_FLAGS) || echo "Builder $(BUILDER_NAME) already exists"
	docker buildx imagetools create -t $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG) $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG)-$(word 1, $(IMAGE_ARCHS)) $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG)-$(word 2, $(IMAGE_ARCHS))
	$(INFO) Image: $(ECR_REPO)/$(PROJECT_NAME)-controller:$(IMAGETAG) manifest list pushed to ECR;
	@$(OK) docker buildx merge complete $(IMAGE)

ifneq ($(filter $(IMAGE_PLATFORM),$(IMAGE_PLATFORMS_LIST)),)
buildx.artifacts.platform: do.buildx.images
else
buildx.artifacts.platform: do.skipx.images
endif
