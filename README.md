# kubernetes wireguard overlay

## About

The wireguard network service runs as a daemonset in a k8s cluster to setup wireguard connections between nodes. Routes are programatically added for `node.spec.PodCIDR` so pod traffic leaving the node will egress through the wg0 interface. Each node will annotate itself with the overlay (wireguard) IP and the wireguard public key. Each service then syncs the local wireguard conf file with the expected state. The expected, or goal, state is determined by checking if the host is configured with the correct `[Interface]` section of the wg0.conf along with every expect `[Peer]`. The peers are the remaining nodes in the cluster and a `list` request is made from each node to see what other nodes are in the cluster and if the service needs to update the config file on the node it's running on.

If a change needs to occur, a new wg0.conf file along with a temporary update file. The presence of this update file is watched by the busybox sidecar which has elevated priveleges to restart the wg0 systemd service on the host.

## Installation

TODO: link to raw yaml for deploying daemonset

## Limitations

TODO
