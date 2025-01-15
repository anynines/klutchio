---
id: creating-a-custom-crossplane-package
title: Creating a Custom Crossplane Package
---

# Creating a Custom Crossplane Package

There are several reasons why you might want to create a custom Crossplane Package. For example:

- Customization: You want to tweak the default setup, for example to set a custom service plan.
- White labeling: You want to rename the API shared by Klutch to reflect your own company name.
- Adding new functionality: You want to add a new, currently unsupported service.

There are two different ways to add a custom package:

1. Adding a new configuration by modifying the default configuration
2. Adding a completely new API

## Modifying the existing Configuration

If you want to make small changes to the default configuration, you can check out the
[crossplane-api developer
guide](https://github.com/anynines/klutchio/blob/main/crossplane-api/README.md). For
white-labeling you should follow the section on adding a new API, and remove the default
configuration.

Once you have made the modifications you want, you can push the configuration package to a OCI image
registry of your choice, and deploy your changes to the control plane cluster by running:

```sh
# Get the name of the configuration
$ kubectl get configurations.pkg.crossplane.io
NAME                             INSTALLED   HEALTHY   PACKAGE                                                AGE
w5n9a2g2-anynines-dataservices   True        True      public.ecr.aws/w5n9a2g2/klutch/dataservices:v1.3.1   3d

# edit the configuration
$ kubectl edit configurations.pkg.crossplane.io w5n9a2g2-anynines-dataservices

```

and then edit `spec.package` to the configuration package you have pushed to your image registry.

## Adding a new API

To add a new API, you need to create a crossplane package with the API you want to add. As a
starting point you can look into the crossplane documentation for Composite Resource Definitions,
Compositions, and Package at (<https://docs.crossplane.io/latest/concepts/>).

Once you have created your custom API and installed it on the Provider cluster, you can add it to
Klutch by following the [steps for adding a custom api to klutch-bind](./adding-custom-service.md)
to make it accessible to your users.
