# kubernetes wireguard overlay

## About

[WireGuard](https://www.wireguard.com/) is a high-performance vpn built into the Linux kernel with simplistic interfaces to setup encrypted connections between two devices quickly. Due to the ease of use, WireGuard can make an efficient mesh network seen in projects like [Tailscale](https://tailscale.com/kb/1151/what-is-tailscale/). This idea can extend to a Kubernetes cluster where the network infrastructure may have limitations that can be circumvented via encapsulation between nodes to create an overlay network for services and pods. In this implementation, nodes are still assigned a PodCIDR which is the subnet for pod IPs on that node locally and will be connected via a bridge, but all traffic leaving the node from the pods will leverage the WireGuard connections between nodes.

This wireguard network service runs as a daemonset in a k8s cluster to setup wireguard connections between nodes. Routes are programatically added for `node.spec.PodCIDR` so pod traffic leaving the node will egress through the wg0 interface. Each node will annotate itself with the overlay (wireguard) IP and the wireguard public key. Each service then syncs the local wireguard conf file with the expected state. The expected, or goal, state is determined by checking if the host is configured with the correct `[Interface]` section of the wg0.conf along with every expected `[Peer]`. The peers are the remaining nodes in the cluster and a `list` request is made from each node to see what other nodes are in the cluster and if the service needs to update the config file on the node it's running on.

If the wireguard systemd service needs to restart to configure an updated peer list, then the daemonset will update the wg0.conf on the host and write an "update" file in the `/etc/wireguard` dir to signal the host should restart `wg-quick@wg0.service`. The presence of this update file is watched by the busybox sidecar which has elevated priveleges to restart the wg0 systemd service on the host.

## Installation

The wireguard network service (wns) daemon and all necessary dependencies can be installed with.

```
kubectl apply -f https://raw.githubusercontent.com/tyler-lloyd/kubernetes-wireguard-overlay/main/specs/wg-network-service.yaml
```

This service's potential is met when the underlying network between nodes has scaling limitations around routing (e.g. route tables in Azure) or it is desired to have cluster traffic be encrypted between nodes over the network.

## Limitations

There are optimizations still to be made re: scale and reliability:

- the daemon should setup a watch on nodes rather than listing nodes on each sync. lacking a watch, this can likely cause problems with massive (> 1000 node) clusters
- potentially many calls to restart the wg0.service which sets up / tears down network links on the host each time -- improve the communication between the daemonset and the host to be able to add peers with the `wg` and `wg-quick` utilities
- currently, it is assumed the node CIDR will not be using a prefix shorter than `/16` and not be in the `100.64.0.0/16` range. Therefore to keep the overlay IP management as simple as possible, the network service keeps the lower 2 octects from the IPv4 node IP and changes the first two to the default overlay subnet mask `100.64`. 
