---
id: oidc-keycloak
title: "Example OpenID Connect (OIDC) Setup: Keycloak"
sidebar_position: 3
tags:
  - configuration
  - documentation
  - OIDC
  - Keycloak
---

# Example OpenID Connect (OIDC) Setup: Keycloak

On this page we describe how to set up Keycloak as an OpenID Connect (OIDC) provider for Klutch.
Please see [Keycloak's documentation](https://www.keycloak.org/documentation) to learn how to deploy
Keycloak.

## OpenID Connect (OIDC) client for the backend

The Klutch backend needs to be configured as an OIDC client so that consumers can authenticate
against it and set up service accounts for [developer (consumer) clusters](./setup-developer-cluster.md)
to connect (bind) to data services. For this purpose, create a new OIDC client for the Klutch
backend. In our example we call the client `klutch-bind-backend`.

![step 1](<keycloak screenshots/Step 1.png>)

Enable _Client authentication_ and _Authorization_, so that Keycloak users can authenticate against
the backend. Select all the flows you want to enable. For the web based setup _Standard Flow_ is
required.

![step 2](<keycloak screenshots/Step 2.png>)

Set up _Root_ and _Home_ URLs as required. For _Valid redirect URLs_ please add
`<BACKEND_URL>/callback`. Replacing `<BACKEND_URL>` with the base URL of the Klutch backend.

![step 3](<keycloak screenshots/Step 3.png>)

## OIDC setting for Users

Currently no special setup is required for users. All users in keycloak can create a binding.
