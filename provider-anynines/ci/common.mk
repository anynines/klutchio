# Common Targets - Buildx and Test workflow

# build releasable artifacts. this will run for each platform being built
buildx.artifacts.platform: ; @:

# build releasable artifacts. this will run once regardless of platform
buildx.artifacts: ; @:

do.buildx.artifacts.%:
	@$(MAKE) buildx.artifacts.platform PLATFORM=$*
do.buildx.artifacts: $(foreach p,$(PLATFORMS), do.buildx.artifacts.$(p))

# helper targets for building multiple platforms
do.buildx.platform.%:
	@$(MAKE) build.check.platform PLATFORM=$*
	@$(MAKE) build.code.platform PLATFORM=$*
do.buildx.platform: $(foreach p,$(PLATFORMS), do.buildx.platform.$(p))

buildx.all:
	@$(MAKE) build.init
	@$(MAKE) build.check
	@$(MAKE) build.code
	@$(MAKE) do.build.platform
	@$(MAKE) build.artifacts
	@$(MAKE) do.buildx.artifacts
	@$(MAKE) buildx.merge
	@$(MAKE) build.done
