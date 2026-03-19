# Default Network Connector Implementation Using a Proxy

## Overview

The default network connector implementation uses an [Envoy Gateway
proxy](https://gateway.envoyproxy.io/) that is deployed on the Klutch Control Plane cluster. This
implementation is designed to work reliably across most infrastructure configurations. It requires
the control plane cluster to be able to create load balancers on the infrastructure via `Service`
Objects with `type: LoadBalancer`. The load balancer by the proxy network connector should be
reachable by app clusters that use the klutch installation. This is the responsibility of the
Operator installing it.

### Architecture

The design is based on a key assumption: **the control plane already has network connectivity to the
automation backend**. This makes it easier for the control plane to also establish connections to
service instances.

It uses the network connector interface, takes in the instance host and port, allocates a port on
the proxy, and writes back the host and port of the proxy to the network connector interface.

The proxy is exposed via a load balancer, new ports allocated on the proxy are also allocated on the
load balancer and the proxy URL is assumed to be the URL of the load balancer. The load balancer can
either have a public URL, or can run on a private network that is accessible to all data service
instances.


# How to install the default network connector

The proxy network connector is based on envoy gateway. To install it, first install the CRDs of the
gateway API. Because the proxy network connector works on a TCP level, the experimental CRDs need to
be installed to expose the TCP connectivity feature.

```sh
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/experimental-install.yaml
```

Next, please install the envoy gateway and create a gateway for the network connector

```sh
helm install envoy-gateway oci://docker.io/envoyproxy/gateway-helm --version v1.5.0 -n envoy-gateway-system --create-namespace -f examples/envoy-gateway/envoygateway-helm-values.yaml

cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy-gateway
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
EOF
```

# Security Features

## IP White-listing

IP White-listing can be used to limit access to data services exposed via the Klutch network
connector on a transport level. This is especially useful if the load balancer used by the network
connector is on a shared or public network.

:::note
IP white-listing requires that the load balancer supports the proxy protocol, so that the network
connector has access to the source IP. By default it would be obscured by the load balancer.
:::

To force enable the proxy protocol please add the following ClientTrafficPolicy:

```yaml
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: ClientTrafficPolicy
metadata:
  name: enable-proxy-protocol
  namespace: a8s-system
spec:
  proxyProtocol:
    optional: false
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: envoy-gateway
```

You can limit the IP ranges which can talk to services via the network connector. To configure a
white-list you need to add a ConfigMap containing the key `allowedCIDRs` with the value being a comma
separated list of IP ranges in CIDR format that are allowed to use the network connector. The IP
ranges need to be the IP ranges the control plane cluster sees when clients try to connect to the
load balancer. If NAT or other technologies are deployed that can obfuscate the client IPs, please
take that into account.

for example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: controller-manager-allowlist
  namespace: a8s-system
data:
  allowedCIDRs: "10.0.0.0/8,fe80::/64"
```
