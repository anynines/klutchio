---
title: TLS Configuration Guide
sidebar_position: 5
tags:
  - TLS
  - Security
  - HTTPS
  - Certificates
  - ProviderConfig
keywords:
  - TLS configuration
  - HTTPS setup
  - Certificate management
  - Self-signed certificates
  - Custom CA certificates
---

# TLS Configuration Guide

This guide provides comprehensive instructions for configuring Transport Layer Security (TLS) in Klutch for secure communication between components, particularly for backup manager clients and service brokers.

**Audience**: End users and Kubernetes operators deploying Klutch in production or development environments.

**Prerequisites**:
- A working Klutch installation with Crossplane running in your cluster
- Access to your backup manager's CA certificate
- Kubectl access to your Kubernetes cluster

## Overview

Klutch supports multiple TLS configuration approaches to accommodate different deployment scenarios:

- **Default**: System CA certificates (production-ready) - no additional configuration needed
- **Development**: Self-signed certificate support with insecure skip verify - for local testing
- **Production**: Custom CA certificate management via Kubernetes secrets - for enterprise deployments

**Backward Compatibility**: Klutch health checks work with both HTTP and HTTPS endpoints. If you have existing HTTP-based backup manager or OSB service endpoints, they will continue to work with the health check system.

## Quick Start

### For Development (Self-Signed Certificates)

This configuration disables certificate verification, allowing your backup manager to use self-signed certificates. Use this only for local development and testing environments.

Save the following YAML to a file (for example, `backup-manager-dev.yaml`):

```yaml
apiVersion: anynines.com/v1
kind: ProviderConfig
metadata:
  name: backup-manager-dev
spec:
  url: https://backup-manager.example.com:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        name: backup-manager-creds
        key: username
    password:
      secretRef:
        name: backup-manager-creds
        key: password
  tls:
    insecureSkipVerify: true
```

Apply it to your cluster:
```
kubectl apply -f backup-manager-dev.yaml
```

### For Production (Custom CA Certificate)

This configuration verifies the backup manager's certificate against your custom CA certificate stored in a Kubernetes secret.

**First, create a Kubernetes secret containing your CA certificate:**

Save this to a file (for example, `ca-secret.yaml`):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backup-manager-ca
  namespace: crossplane-system
type: Opaque
data:
  ca.crt: <base64-encoded-certificate-content>
```

Apply it:
```
kubectl apply -f ca-secret.yaml
```

**Then, create a ProviderConfig that references this secret:**

Save this to a file (for example, `backup-manager-prod.yaml`):

```yaml
apiVersion: anynines.com/v1
kind: ProviderConfig
metadata:
  name: backup-manager-prod
spec:
  url: https://backup-manager.example.com:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        name: backup-manager-creds
        key: username
    password:
      secretRef:
        name: backup-manager-creds
        key: password
  tls:
    insecureSkipVerify: false
    caBundleSecretRef:
      name: backup-manager-ca
      namespace: crossplane-system
      key: ca.crt
```

Apply it:
```
kubectl apply -f backup-manager-prod.yaml
```

## Configuration Approaches

### Approach 1: Backward Compatible (Default)

When no TLS settings are specified, Klutch uses system CA certificates automatically. This is the default behavior and requires no additional configuration.

**Use Case**: When your backup manager uses certificates signed by standard CAs (like Let's Encrypt or other public CAs).

**Backward Compatibility**: Klutch health checks also support HTTP endpoints automatically. If you're using HTTP URLs for backup manager or OSB services, the health check will attempt to connect without requiring HTTPS. This ensures existing deployments continue to work without modification.

### Approach 2: Development (Insecure Skip Verify)

Set `insecureSkipVerify: true` to bypass certificate verification. This allows self-signed certificates for development and testing environments.

**Use Case**: Local development, testing with self-signed certificates.

⚠️ **Warning**: Never use `insecureSkipVerify: true` in production!

### Approach 3: Production (Custom CA)

Use `caBundleSecretRef` to validate certificates against a custom CA bundle stored in a Kubernetes secret. This is the recommended approach for production.

**Use Case**: Production deployments with internal CAs or custom certificate hierarchies.

## ProviderConfig TLS Settings

### Configuration Fields

The `tls` section in your ProviderConfig controls how Klutch validates certificates when connecting to the backup manager:

```yaml
spec:
  tls:
    # Skip certificate verification (boolean)
    # Set to true only for development/testing with self-signed certificates
    insecureSkipVerify: false

    # Reference to a Kubernetes secret containing your CA certificate(s)
    # This secret stores the certificate file that will be used to verify connections
    caBundleSecretRef:
      # Name of the Kubernetes secret
      name: backup-manager-ca

      # Kubernetes namespace where the secret is stored
      namespace: crossplane-system

      # Key within the secret that contains the certificate data
      # This defaults to "ca.crt" if not specified
      key: ca.crt
```

### Creating a Secret with CA Certificate

To create a Kubernetes secret with your CA certificate, you need to base64-encode the certificate content.

#### Single Certificate

If you have a single CA certificate file (for example, `ca.crt`), encode it and save to a file:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backup-manager-ca
  namespace: crossplane-system
type: Opaque
data:
  ca.crt: <base64-encoded-certificate-content>
```

Replace `<base64-encoded-certificate-content>` with the base64-encoded content of your certificate file.

#### Multiple Certificates (CA Bundle)

If you have multiple CA certificates (for example, from a certificate chain), concatenate them and base64-encode the combined content:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backup-manager-ca
  namespace: crossplane-system
type: Opaque
data:
  ca.crt: <base64-encoded-certificate-chain>
```

The certificate chain should contain all certificates concatenated together, each starting with `-----BEGIN CERTIFICATE-----` and ending with `-----END CERTIFICATE-----`.

## Certificate Formats

### Supported Format: PEM

Klutch supports certificates in PEM (Privacy Enhanced Mail) format, which is the most common format for TLS certificates.

A PEM certificate is a text file that looks like this:

```
-----BEGIN CERTIFICATE-----
MIICljCCAX4CCQCKz0Td7gbqnDANBgkqhkiG9w0BAQsFADANMQswCQYDVQQGEwJV
... base64 encoded certificate data ...
-----END CERTIFICATE-----
```

**How to verify your certificate format**: Open your certificate file in a text editor. If you see the `-----BEGIN CERTIFICATE-----` and `-----END CERTIFICATE-----` markers, it's in PEM format and ready to use with Klutch.

**How to base64-encode your certificate**: Take the entire contents of your PEM certificate file (including the BEGIN and END markers) and convert it to base64 format. Most tools and documentation will show how to do this on your specific operating system.

## Usage Scenarios

### Scenario 1: Production with Enterprise CA

**Situation**: Your organization uses an internal Certificate Authority to sign the backup manager's certificate. You need to configure Klutch to trust this enterprise CA.

**What you need**:
- Your enterprise CA certificate file (usually provided by your IT/Security team)

**Steps**:

1. Create a Kubernetes secret with your enterprise CA certificate:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: enterprise-ca
  namespace: crossplane-system
type: Opaque
data:
  ca.crt: <base64-encoded-enterprise-ca-certificate>
```

2. Create a ProviderConfig that references this CA:

```yaml
apiVersion: anynines.com/v1
kind: ProviderConfig
metadata:
  name: backup-manager-prod
spec:
  url: https://backup-manager.internal.company.com:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        name: backup-manager-creds
        key: username
    password:
      secretRef:
        name: backup-manager-creds
        key: password
  tls:
    insecureSkipVerify: false
    caBundleSecretRef:
      name: enterprise-ca
      namespace: crossplane-system
```

The backup manager's certificate must be issued by this enterprise CA for the connection to succeed.

### Scenario 2: Development with Self-Signed Certificate

**Situation**: You're testing locally with a backup manager that uses a self-signed certificate. For development purposes, you can disable certificate verification.

**Important**: Only use this configuration in development. Never use `insecureSkipVerify: true` in production.

```yaml
apiVersion: anynines.com/v1
kind: ProviderConfig
metadata:
  name: backup-manager-dev
spec:
  url: https://backup-manager.local:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        name: backup-manager-creds
        key: username
    password:
      secretRef:
        name: backup-manager-creds
        key: password
  tls:
    insecureSkipVerify: true
```

This configuration bypasses certificate validation entirely, allowing connections to any HTTPS endpoint regardless of certificate validity.

### Scenario 3: Multiple Backup Managers with Different CAs

**Situation**: Your organization runs multiple backup manager instances, each signed by a different CA. You need separate configurations for each.

**What you need**:
- CA certificate for each backup manager instance

**Steps**:

1. Create a secret for each CA:

```yaml
# Production CA
apiVersion: v1
kind: Secret
metadata:
  name: backup-manager-ca-prod
  namespace: crossplane-system
type: Opaque
data:
  ca.crt: <base64-encoded-prod-ca-certificate>
---
# Staging CA
apiVersion: v1
kind: Secret
metadata:
  name: backup-manager-ca-staging
  namespace: crossplane-system
type: Opaque
data:
  ca.crt: <base64-encoded-staging-ca-certificate>
```

2. Create a ProviderConfig for production:

```yaml
apiVersion: anynines.com/v1
kind: ProviderConfig
metadata:
  name: backup-manager-prod
spec:
  url: https://backup-manager.prod.company.com:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        name: backup-manager-creds-prod
        key: username
    password:
      secretRef:
        name: backup-manager-creds-prod
        key: password
  tls:
    insecureSkipVerify: false
    caBundleSecretRef:
      name: backup-manager-ca-prod
      namespace: crossplane-system
```

3. Create a ProviderConfig for staging:

```yaml
apiVersion: anynines.com/v1
kind: ProviderConfig
metadata:
  name: backup-manager-staging
spec:
  url: https://backup-manager.staging.company.com:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        name: backup-manager-creds-staging
        key: username
    password:
      secretRef:
        name: backup-manager-creds-staging
        key: password
  tls:
    insecureSkipVerify: false
    caBundleSecretRef:
      name: backup-manager-ca-staging
      namespace: crossplane-system
```

Now your Backup and Restore resources can reference the appropriate ProviderConfig for each environment.

## Security Best Practices

### ✅ DO
- Use `insecureSkipVerify: false` in all production environments
- Store CA certificates in Kubernetes secrets
- Use HTTPS (https://) URLs for all connections
- Rotate certificates before expiration
- Verify certificate hostname matches the service FQDN
- Use strong key sizes (RSA 2048 bits minimum, 4096 recommended)
- Audit certificate access and modifications
- Document certificate hierarchy and rotation procedures

### ❌ DON'T
- Deploy `insecureSkipVerify: true` to production
- Hardcode certificates in application code
- Use HTTP URLs for sensitive services
- Disable certificate verification permanently
- Share CA private keys
- Store secrets without RBAC controls
- Use self-signed certificates in production without understanding the risks
- Ignore certificate expiration warnings

## Troubleshooting

### Issue: "x509: certificate signed by unknown authority"

**What this means**: Klutch cannot verify the backup manager's certificate because it doesn't recognize the Certificate Authority that signed it.

**Solution 1: For development only - disable certificate verification**

Set `insecureSkipVerify: true` in your ProviderConfig. This allows any certificate to be used:

```yaml
spec:
  tls:
    insecureSkipVerify: true
```

Only use this in development or testing environments.

**Solution 2: For production - provide the CA certificate**

Create a Kubernetes secret with your CA certificate and reference it in your ProviderConfig:

```yaml
spec:
  tls:
    insecureSkipVerify: false
    caBundleSecretRef:
      name: backup-manager-ca
      namespace: crossplane-system
```

### Issue: "failed to parse CA certificate"

**What this means**: The certificate data in your Kubernetes secret is not in a valid format that Klutch can read.

**How to fix**:

1. Verify your certificate is in PEM format
2. Ensure the base64-encoded content is correct
3. Check that all BEGIN/END markers are properly included
4. Try creating the secret again with valid certificate data

### Issue: "TLS handshake failure"

**What this means**: Klutch cannot establish a secure connection to the backup manager. This usually means the certificate doesn't match the hostname you're connecting to.

**Check these things**:

- **Verify the URL matches the certificate**: If your ProviderConfig URL is `https://backup-manager.internal`, the certificate must be for `backup-manager.internal`
- **Check certificate expiration**: Make sure your certificate hasn't expired
- **Verify you're using the correct secret**: Double-check that your `caBundleSecretRef` points to the correct secret
- **Confirm the secret exists**: Make sure the secret you referenced actually exists in the `crossplane-system` namespace

### Issue: "Connection refused" or "Connection timeout"

**What this means**: Klutch cannot connect to the backup manager at all. This is usually a network issue, not a certificate issue.

**Check these things**:

- **Verify the URL is correct**: Make sure the URL, hostname, and port are correct in your ProviderConfig
- **Check network connectivity**: Verify that the backup manager is reachable from your Kubernetes cluster
- **Check firewall rules**: Ensure firewalls allow connections from your cluster to the backup manager
- **Verify the backup manager is running**: Confirm the backup manager service is actually running and listening on the specified port

### Cryptographic Support

Klutch uses Go's standard cryptographic libraries for all TLS operations. This means:

- **TLS 1.2 minimum** - Older TLS versions are not supported
- **Standard certificate validation** - Industry-standard certificate chain validation
- **No external dependencies** - Uses only Go standard library, reducing security risks

### What Happens During Certificate Validation

When `insecureSkipVerify: false`, Klutch performs these checks:

1. **Signature verification** - Confirms the certificate was signed by a trusted CA
2. **Chain validation** - Verifies the entire certificate chain from your CA to the server certificate
3. **Hostname verification** - Ensures the certificate's hostname matches your URL
4. **Expiration check** - Confirms the certificate hasn't expired
5. **Key usage validation** - Verifies the certificate is allowed for this type of connection

## Migration from Unencrypted to Encrypted

### Overview

This guide walks you through updating your ProviderConfigs to use encrypted HTTPS connections with proper certificate validation instead of unencrypted HTTP.

### Step 1: Prepare Your CA Certificate

Obtain your CA certificate in PEM format. This is the certificate that signed your backup manager's certificate. Usually, your IT or security team can provide this.

### Step 2: Create the Kubernetes Secret

Create a Kubernetes secret in the `crossplane-system` namespace containing your CA certificate. Save this to a file and apply it:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: backup-manager-ca
  namespace: crossplane-system
type: Opaque
data:
  ca.crt: <base64-encoded-certificate-content>
```

### Step 3: Update Your ProviderConfig

Update your existing ProviderConfig to use HTTPS and reference your new secret:

```yaml
apiVersion: anynines.com/v1
kind: ProviderConfig
metadata:
  name: backup-manager
spec:
  url: https://backup-manager.example.com:3000
  providerCredentials:
    source: Secret
    username:
      secretRef:
        name: backup-manager-creds
        key: username
    password:
      secretRef:
        name: backup-manager-creds
        key: password
  tls:
    insecureSkipVerify: false
    caBundleSecretRef:
      name: backup-manager-ca
      namespace: crossplane-system
      key: ca.crt
```

Key changes:
- Change `url` from `http://` to `https://`
- Add the `tls` section with your CA certificate reference

### Step 4: Verify Connectivity

After updating your ProviderConfig, test that Klutch can connect to the backup manager. Try creating a test backup resource to verify everything works:

```yaml
apiVersion: dataservices.anynines.com/v1
kind: Backup
metadata:
  name: test-backup
spec:
  providerConfigRef:
    name: backup-manager
  # ... rest of your backup configuration
```

If the backup succeeds, your TLS configuration is correct.

## Additional Resources

- [Kubernetes Secrets Documentation](https://kubernetes.io/docs/concepts/configuration/secret/) - How to work with secrets in Kubernetes
- [Crossplane Documentation](https://docs.crossplane.io/) - More information about Crossplane and ProviderConfigs

## Support and Questions

For issues or questions about TLS configuration:

1. Check the [Troubleshooting](#troubleshooting) section
2. Verify your certificate is in valid PEM format
3. Ensure your secret is created in the correct namespace (`crossplane-system`)
4. Check that your ProviderConfig is correctly referencing the secret
5. Consult your organization's certificate management or security team
