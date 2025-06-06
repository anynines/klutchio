# Klutch

## Description

Klutch extends [Crossplane](https://www.crossplane.io/) to manage resources across multiple
Kubernetes clusters. More details about Klutch's
[core concepts can be found here](https://klutch.io/docs/architecture/).

Klutch makes it possible to share Kubernetes API-driven services of any `Kind` across a large number
of clusters. Service providers remain in control of operations, service users can focus on using the
service and only need a lightweight adapter in their cluster.

## Getting started

Please refer to our documentation [for platform operators](https://klutch.io/docs/platform-operator/)
if you want to enable your users to use various managed resources. If you're a software developer
then please read [our documentation for developers](https://klutch.io/docs/developer-guide/).

## Getting Involved

If you want to contact the Klutch authors or you're looking for support, please read
[this](https://klutch.io/docs/community).

## What can Klutch do for you?

### User stories

**Internal Platform**

Kubernetes based internal platform projects have two choices - sharing clusters amongst teams or
operating many isolated clusters. When sharing clusters across teams, permissions need to be locked
down to ensure that teams don't interfere with each other. As a downside this limits customization
possibilities for the teams. Custom Resource Definitions are cluster scoped and thus must to be
shared by users of an individual cluster.

In environments where many Kubernetes clusters need to be served with the same APIs usually
[Operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) are used. Often this
leads to sprawl since operators must be installed on every cluster where the API is needed. This can
make tasks like monitoring, patch management and cost control significantly more challenging.

With Klutch, all the operations of the service can be bundled, and only the Custom Resources are
visible to the users. The impact on users' freedom is minimal, they can use centrally provided
services, for example databases or identity providers.

**SaaS provider**

A Software-As-A-Service (SaaS) provider might already be using Kubernetes to run and manage their
infrastructure and provide that to their users via a public API. For products and services for
developers the same customers may also want to use their service via a Kubernetes API,so they can
leverage tools of the CNCF landscape. Traditionally, the provider would create an Operator or
Crossplane provider that exposes a
[Custom Resource Definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/),
and talks to the provider's API which creates a custom resource on the provider's infrastructure.
This seems like a lot of overhead and it is. With Klutch you can take your existing CRD based
automation, and share it with users that bring their own Kubernetes cluster.

Advantages include:

- less overhead
- a trusted standard which users can rely on across different services

### What can Klutch do Today

- Bring your [Open Service Broker API](https://www.openservicebrokerapi.org/) based automation to
  Kubernetes
- Provides you with central management for your Crossplane definitions
- Make available your Operator based service to users in other clusters

## Developing

1. Init go.work

   Execute the following commands when the repository is freshly cloned or if you are getting import
   errors in your go module. The commands enable go to know which module you're working on if you
   have multiple modules in the same workspace. For more details please refer to the official
   [go.work docs].

   ```bash
   go work init ./provider-anynines
   go work use ./clients/a9s-backup-manager
   go work use ./clients/a9s-open-service-broker
   ```

[go.work docs]: https://go.dev/doc/tutorial/workspaces

## Documentation Workflow

We use GitHub Actions to automate our documentation process:

- Pull requests modifying `docs/` trigger preview builds.
- Previews are deployed to PR-specific URLs for easy review.
- Merges to main automatically update the production documentation.
- Preview environments are cleaned up after PR merges.

When contributing to documentation:

1. Create a PR with your changes.
2. Check the automatically posted comment for the preview link.
3. Review your changes in the preview environment.
4. After approval and merge, changes will go live on the main site.

## Credits

- The Klutch Authors
