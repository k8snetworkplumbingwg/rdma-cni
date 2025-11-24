[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/k8snetworkplumbingwg/rdma-cni)](https://goreportcard.com/report/github.com/k8snetworkplumbingwg/rdma-cni)
[![Build&Tests](https://github.com/k8snetworkplumbingwg/rdma-cni/actions/workflows/buildtest.yaml/badge.svg?branch=master)](https://github.com/k8snetworkplumbingwg/rdma-cni/actions/workflows/buildtest.yaml)
[![Coverage Status](https://coveralls.io/repos/github/k8snetworkplumbingwg/rdma-cni/badge.svg)](https://coveralls.io/github/k8snetworkplumbingwg/rdma-cni)

# RDMA CNI plugin
CNI compliant plugin for network namespace aware RDMA interfaces.

RDMA CNI plugin allows network namespace isolation for RDMA workloads in a containerized environment.

# Overview
RDMA CNI plugin is intended to be run as a chained CNI plugin (introduced in [CNI Specifications `v0.3.0`](https://github.com/containernetworking/cni/blob/v0.3.0/SPEC.md#network-configuration)).
It ensures isolation of RDMA traffic from other workloads in the system by moving the associated RDMA interfaces of the
provided network interface to the container's network namespace path.

The main use-case (for now...) is for containerized SR-IOV workloads orchestrated by [Kubernetes](https://kubernetes.io/)
that perform [RDMA](https://community.mellanox.com/s/article/what-is-rdma-x) and wish to  leverage network namespace
isolation of RDMA devices introduced in linux kernel `5.3.0`.

# Requirements

## Hardware
SR-IOV capable NIC which supports RDMA.

### Supported Hardware

#### Mellanox Network adapters
ConnectX®-4 and above

## Operating System
Linux distribution

### Kernel
Kernel based on `5.3.0` or newer, RDMA modules loaded in the system.
[`rdma-core`](https://github.com/linux-rdma/rdma-core) package provides means to automatically load relevant modules
on system start.

> __*Note:*__ For deployments that use Mellanox out-of-tree driver (Mellanox OFED), Mellanox OFED version `4.7` or newer
>is required. In this case it is not required to use a Kernel based on `5.3.0` or newer.

### Pacakges
[`iproute2`](https://mirrors.edge.kernel.org/pub/linux/utils/net/iproute2/) package based on kernel `5.3.0` or newer
installed on the system.

> __*Note:*__ It is recommended that the required packages are installed by your system's package manager.

> __*Note:*__ For deployments using Mellanox OFED, `iproute2` package is bundled with the driver under `/opt/mellanox/iproute2/`

## Deployment requirements (Kubernetes)
Please refer to the relevant link on how to deploy each component.
For a Kubernetes deployment, each SR-IOV capable worker node should have:

- [SR-IOV network device plugin](https://github.com/intel/sriov-network-device-plugin) deployed and configured with an [RDMA enabled resource](https://github.com/intel/sriov-network-device-plugin/tree/master/docs/rdma)
- [Multus CNI](https://github.com/intel/multus-cni) `v3.4.1` or newer deployed and configured
- Per fabric SR-IOV supporting CNI deployed
    - __Ethernet__: [SR-IOV CNI](https://github.com/intel/sriov-cni)

> __*Note:*__: Kubernetes version 1.16 or newer is required for deploying as daemonset

# RDMA CNI configurations
```json
{
  "cniVersion": "0.3.1",
  "type": "rdma",
  "args": {
    "cni": {
      "debug": true
    }
  }
}
```
> __*Note:*__ "args" keyword is optional.

# Deployment

## System configuration
It is recommended to set RDMA subsystem namespace awareness mode to `exclusive` on OS boot.

Set RDMA subsystem namespace awareness mode to `exclusive` via `ib_core` module parameter:
```console
~$ echo "options ib_core netns_mode=0" >> /etc/modprobe.d/ib_core.conf
```

Set RDMA subsystem namespace awareness mode to `exclusive` via rdma tool:
```console
~$ rdma system set netns exclusive
```

> __*Note:*__ When changing RDMA subsystem netns mode, kernel requires that no network namespaces to exist in the system.

## Deploy RDMA CNI
```bash
$ kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/rdma-cni/refs/tags/v1.5.0/deployment/rdma-cni-daemonset.yaml
```

## Deploy workload
Pod definition can be found in the example below.
```bash
$ kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/rdma-cni/refs/tags/v1.5.0/examples/rdma_test_pod.yaml
```

## Example resource

- [Pod](./examples/my_rdma_test_pod.yaml): test workload
- [SR-IOV Network Device Plugin ConfigMap](./examples/sriov_dp_rdma_resource.yaml): defines an RDMA enabled SR-IOV resource pool named: mellanox.com/sriov_rdma
- [Network CRD](./examples/rdma_net_crd.yaml):  defines a network, `sriov-network`, associated with an rdma enabled resurce, `mellanox.com/sriov_rdma`. The CNI plugins that will be executed in a chain are for Pods that request this network are: _sriov_, _rdma_ CNIs

# Development
It is recommended to use the same go version as defined in `.travis.yml`
to avoid potential build related issues during development (newer version will most likely work as well).

### Build from source
```bash
$ git clone https://github.com/k8snetworkplumbingwg/rdma-cni.git
$ cd rdma-cni
$ make
```
Upon a successful build, `rdma` binary can be found under `./build`.
For small deployments (e.g a kubernetes test cluster/AIO K8s deployment) you can:
1. Copy `rdma` binary to the CNI dir in each worker node.
2. Build container image, push to your own image repo then modify the deployment template and deploy. 

#### Run tests:
```bash
$ make tests
```

#### Build image:
```bash
$ make image
```

# Limitations
## Ethernet
### RDMA workloads utilizing RDMA Connection Manager (CM)
For Mellanox Hardware, due to kernel limitation, it is required to pre-allocate MACs for all VFs in the deployment
if an RDMA workload wishes to utilize RMDA CM to establish connection.

This is done in the following manner:

__Set VF administrative MAC address :__

```
$ ip link set <pf-netdev> vf <vf-index> mac <mac-address>
```

__Unbind/Bind VF driver :__

```
$ echo <vf-pci-address> > /sys/bus/pci/drivers/mlx5_core/unbind
$ echo <vf-pci-address> > /sys/bus/pci/drivers/mlx5_core/bind
```

__Example:__
```
$ ip link set enp4s0f0 vf 3 mac 02:03:00:00:48:56
$ echo 0000:03:00.5 > /sys/bus/pci/drivers/mlx5_core/unbind
$ echo 0000:03:00.5 > /sys/bus/pci/drivers/mlx5_core/bind
```
Doing so will populate the VF's node and pord GUID required for RDMA CM to establish connection.
