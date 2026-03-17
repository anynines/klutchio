# Release Process for Klutch Bind

1. Build and publish the konnector image:

   First, login to ECR and then run the following command (replace `vx.y.z` with the version you are releasing):

   - Set up the ECR profile such that it points to dedicated ECR AWS account. You can find the account on the AWS access portal page.

   1. Set the AWS profile name used for this release session:

      ```bash
      export AWS_PROFILE=ECR
      ```

   2. Configure credentials for that profile (if not already configured):

      ```bash
      aws configure --profile=ECR
      ```

   3. Authenticate Docker with Amazon ECR Public:

      ```bash
      aws ecr-public get-login-password --region us-east-1 --profile=ECR | docker login --username AWS --password-stdin public.ecr.aws/w5n9a2g2
      ```

   4. Build and publish the multi-architecture konnector image:

      ```bash
      KO_DOCKER_REPO=public.ecr.aws/w5n9a2g2/klutch/konnector ko build ./cmd/konnector --bare -t vx.y.z --platform=linux/amd64,linux/arm64
      ```

2. Build and publish the example-backend image:

   Run the following command (replace `vx.y.z` with the version you are releasing):

   Build and publish the multi-architecture example-backend image:

   ```bash
   KO_DOCKER_REPO=public.ecr.aws/w5n9a2g2/klutch/example-backend ko build ./cmd/example-backend --bare -t vx.y.z --platform=linux/amd64,linux/arm64
   ```

3. Kubectl-bind is used from klutch enterprise.
