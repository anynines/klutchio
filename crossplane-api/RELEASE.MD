# Release Process for Crossplane Components

1. Login to ECR:

   ```bash
   aws ecr-public get-login-password --region us-east-1 --profile=ECR | docker login --username AWS --password-stdin public.ecr.aws/w5n9a2g2
   ```

2. Build and Push Package:

    **Data Services:**
    Run the following command (replace `vX.Y.Z` with the version you are releasing for a9s Data Services):

    ```bash
    make -C crossplane-api/ dataservices-config-push dataservicesConfigVersion=vX.Y.Z
    ```

    **Provider Anynines:**
    Run the following command (replace `vX.Y.Z` with the version you are releasing for provider-anynines):

    ```bash
    cd provider-anynines
    make provider-controller-push IMAGETAG=vX.Y.Z
    make provider-build-push IMAGETAG=vX.Y.Z
    ```

3. Update documented version of components:

   1. Update the configuration packages in the [`deploy`](./deploy) folder to use the new versions you just pushed.
   2. Update the [`README.md`](./README.md) to use the new versions you just pushed.
   3. **Important:** Ensure all providers and functions versions are synced and consistent across all configuration files.
