# RDMA CNI plugin
CNI compliant plugin for RDMA namespace aware network interfaces

# Overview
This software will enable namespace isoloation for RDMA workloads on a kubernetes
cluster.

## At high level
RDMA CNI plugin is intended to be run as a chained CNI plugin (introduced in CNI Specifications `v0.3.0`) moving the associated
RDMA interfaces of a given network interface to the provided network namespace path

## Links
https://github.com/containernetworking/cni/blob/v0.3.0/SPEC.md#network-configuration
