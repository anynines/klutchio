The dockerfile in this repo is used by the build-and-bump-git-dockerfile-image-run pipeline.
Any tools required to build the provider need to be added to the Dockerfile.
The ecr image output by the build-and-bump pipeline is used by build-crossplane-provider-pipeline-run pipeline to build this repo.