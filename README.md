# kubernetes wireguard overlay

## About

[WireGuard](https://www.wireguard.com/) is a [high-performance](https://www.wireguard.com/performance/) vpn built into the Linux kernel with simplistic interfaces to setup encrypted connections between two devices quickly. Due to the ease of use, WireGuard can make an efficient mesh network seen in projects like [Tailscale](https://tailscale.com/kb/1151/what-is-tailscale/). This idea can extend to a Kubernetes cluster where the network infrastructure may have limitations that can be circumvented via encapsulation between nodes to create an overlay network for services and pods. In this implementation, nodes are still assigned a PodCIDR which is the subnet for pod IPs on that node locally and will be connected via a bridge, but all traffic leaving the node from the pods will leverage the WireGuard connections between nodes.

WireGuard interfaces are setup and maintained with a dance between `wg(8)` and `ip(8)`. The WireGuard utility `wg` is responsible for setting and retrieving the current state of WireGuard devices on a host, while `ip` handles typical Linux networking components needed for the WireGuard overlay to work, namely networking devices (`ip-link(8)`) and routes. For performing the steps of this dance, this project uses [wgctrl](https://pkg.go.dev/golang.zx2c4.com/wireguard/wgctrl) for interacting with `wg` and [vishvananda/netlink](https://pkg.go.dev/github.com/vishvananda/netlink) for ensuring the link and routes are appropriately set on a host. **Note:** under the hood, wgctrl uses [mdlayher/netlink](https://pkg.go.dev/github.com/mdlayher/netlink) as its `netlink(7)` library.

In an effort to maximize performance, the WireGuard state on every node is handled in software and no data is persisted to disk (keys or config files). This also means there is an absence of using `wg-quick(8)` for bringing up or re-syncing a config for the WireGuard interface. Once a node joins the Kubernetes cluster, it will annotate itself with its WireGuard public key and IP and add all other nodes in the cluster as WireGuard peers. Once the newly added node annotates itself, the other nodes receive this update event and can add the node as peer. Similarly, once a node is deleted the remaining nodes are notified and remove the node from their list of peers.

Because the overlay network exists as connections between nodes, each node in the cluster needs to become a WireGuard device and then maintain the state of the overlay by watching other nodes as they're created/updated/deleted. Therefore, this service runs as a daemonset in a cluster. Each node has an agent to ensure the local WireGuard state is configured correctly. There are two components to accomplish this:

### WireGuard Initializer (wg-init)

wg-init runs as an init container and is responsible for

- A device on the host of type `wireguard` exists and has the right overlay IP attached to the interface
  - this is equivilent of running `ip link add dev wg0 type wireguard && ip addr add dev wg0 $OVERLAY_IP`
- Routes exist on the host which send on all traffic from the PodCIDRs and OverlayCIDR out the WireGuard device (default `wg0`)

### WireGuard Network Controller (wnc)

The WireGuard Network Controller runs on each node and watches all other nodes, waiting to either add a new node as a peer or delete a peer if a node goes down. The controller uses the `controller-runtime` pkg to prevent overloading the API service in large clusters.

## Installation

The WireGuard Network Controller (wnc) and all necessary dependencies can be installed with.

```
kubectl apply -f https://raw.githubusercontent.com/tyler-lloyd/kubernetes-wireguard-overlay/main/specs/wg-network-service.yaml
```

This service's potential is met when the underlying network between nodes has scaling limitations around routing (e.g. route tables in Azure) or it is desired to have cluster traffic be encrypted between nodes over the network.

## Limitations/TODO

- To keep the overlay IPAM as simple as possible, each node's WireGuard IP is computed by taking the lower two octets of the `spec.hostIP` and merging it with the mask of `100.64.0.0/16` by default. It is assumed the node CIDR will not be using a prefix shorter than `/16` and not be in the `100.64.0.0/16` range.
- Does not remove reference to old peer if a key is rotated. If a node rotates its keys then all nodes will add this as a new peer, leaving a stale reference in its list of peers. The new peer will still work as expected as the endpoint will move to the new public key.
- No cleanup of peers that do no exist in the cluster. Each node assumes the metadata for every peer is in the cluster via annotations on the nodes or will eventually be added for new nodes. There could be a case where a node is deleted and a controller on another node crashes at the same time which would prevent the deleted node from being removed as a peer.
- Logging in the controller is leftover from kubebuilder and is not in a great format (and continues to have development mode turned on).
- Still lingering init-container for installing wireguard if needed (assumed apt based package management). Won't be needed for any systems using kernel 5.6+ or have already installed wireguard on their hosts.
