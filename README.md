# kubernetes wireguard overlay

## About

[WireGuard](https://www.wireguard.com/) is a [high-performance](https://www.wireguard.com/performance/) vpn built into the Linux kernel with simplistic interfaces to setup encrypted connections between two devices quickly. Due to the ease of use, WireGuard can make an efficient mesh network seen in projects like [Tailscale](https://tailscale.com/kb/1151/what-is-tailscale/). This idea can extend to a Kubernetes cluster where the network infrastructure may have limitations that can be circumvented via encapsulation between nodes to create an overlay network for services and pods. In this implementation, nodes are still assigned a PodCIDR which is the subnet for pod IPs on that node locally and will be connected via a bridge, but all traffic leaving the node from the pods will leverage the WireGuard connections between nodes.

WireGuard interfaces are setup and maintained with a dance between `wg(8)` and `ip(8)`. The WireGuard utility `wg` is responsible for setting and retrieving the current state of WireGuard devices on a host, while `ip` handles typical Linux networking components needed for the WireGuard overlay to work, namely networking devices (`ip-link(8)`) and routes. For performing the steps of this dance, this project uses [wgctrl](https://pkg.go.dev/golang.zx2c4.com/wireguard/wgctrl) for interacting with `wg` and [vishvananda/netlink](https://pkg.go.dev/github.com/vishvananda/netlink) for ensuring the link and routes are appropriately set on a host. **Note:** under the hood, wgctrl uses [mdlayher/netlink](https://pkg.go.dev/github.com/mdlayher/netlink) as its `netlink(7)` library.

In an effort to maximize performance, the WireGuard state on every node is handled in software and no data is persisted to disk (keys or config files). This also means there is an absence of using `wg-quick(8)` for bringing up or re-syncing a config for the WireGuard interface. Once a node joins the Kubernetes cluster, it will annotate itself with its WireGuard public key and IP and add all other nodes in the cluster as WireGuard peers. Once the newly added node annotates itself, the other nodes will see this update and add the new node as a new peer. Similarly, once a node is deleted then all other nodes see this delete operation and remove the node as a WireGuard peer.

Because the overlay network exists as connections between nodes, each node in the cluster needs to become a WireGuard device and then maintain the state of the overlay by watching other nodes as they're created/updated/deleted. Therefore, this service runs as a daemonset on a cluster, ensuring each node has an agent to ensure the local WireGuard state is configured correctly. There are two components to accomplish this:

### WireGuard Initializer (wg-init)

wg-init runs as an init container and is responsible for

- A device on the host of type `wireguard` exists and has the right overlay IP attached to the interface
  - this is equivilent of running `ip link add dev wg0 type wireguard && ip addr add dev wg0 $OVERLAY_IP`
- Routes exist on the host which send on all traffic from the PodCIDRs and OverlayCIDR out the WireGuard device (default `wg0`)

### WireGuard Network Controller (wnc)

The WireGuard Network Controller runs on each node and watches all other nodes, waiting to either add a new node as a peer or delete a peer if a node goes down. The controller uses the `controller-runtime` pkg to prevent overloading the API service in large clusters.

## Installation

The wireguard network service (wns) daemon and all necessary dependencies can be installed with.

```
kubectl apply -f https://raw.githubusercontent.com/tyler-lloyd/kubernetes-wireguard-overlay/main/specs/wg-network-service.yaml
```

This service's potential is met when the underlying network between nodes has scaling limitations around routing (e.g. route tables in Azure) or it is desired to have cluster traffic be encrypted between nodes over the network.

## Limitations

- currently, it is assumed the node CIDR will not be using a prefix shorter than `/16` and not be in the `100.64.0.0/16` range. Therefore to keep the overlay IP management as simple as possible, the network service keeps the lower 2 octects from the IPv4 node IP and changes the first two to the default overlay subnet mask `100.64`. 
