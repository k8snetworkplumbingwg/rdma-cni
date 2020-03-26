[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mellanox/rdma-cni)](https://goreportcard.com/report/github.com/Mellanox/rdma-cni)
[![Build Status](https://travis-ci.com/Mellanox/rdma-cni.svg?branch=master)](https://travis-ci.com/Mellanox/rdma-cni)
[![Coverage Status](https://coveralls.io/repos/github/Mellanox/rdma-cni/badge.svg)](https://coveralls.io/github/Mellanox/rdma-cni)

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
>is required

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
    - __Infiniband__: [Infiniband SR-IOV CNI](https://github.com/Mellanox/ib-sriov-cni)

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
Set RDMA subsystem namespace awareness mode to `exclusive`
```console
~$ rdma system set netns exclusive
```

## Deploy RDMA CNI
```console
~$ kubectl apply -f ./deployment/rdma-cni-daemonset.yaml
```

## Deploy workload
Pod definition can be found in the example below.
```console
~$ kubectl apply -f ./examples/my_rdma_test_pod.yaml
```

### Pod example:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: rdma-test-pod
  annotations:
    k8s.v1.cni.cncf.io/networks: sriov-rdma-net
spec:
  containers:
    - name: rdma-app
      image: centos/tools
      imagePullPolicy: IfNotPresent
      command: [ "/bin/bash", "-c", "--" ]
      args: [ "while true; do sleep 300000; done;" ]
      resources:
        requests:
          mellanox.com/sriov_rdma: '1'
        limits:
          mellanox.com/sriov_rdma: '1'
```

## SR-IOV Network Device Plugin ConfigMap example
The following `yaml` defines an RDMA enabled SR-IOV resource pool named: `mellanox.com/sriov_rdma`
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: sriovdp-config
  namespace: kube-system
data:
  config.json: |
    {
      "resourceList": [
        {
           "resourcePrefix": "mellanox.com",
           "resourceName": "sriov_rdma",
           "isRdma": true,
           "selectors": {
               "vendors": ["15b3"],
               "pfNames": ["enp4s0f0"]
           }
        }
      ]
    }
```

## Network CRD example
The following `yaml` defines a network, `sriov-network`, associated with an rdma enabled resurce, `mellanox.com/sriov_rdma`.

The CNI plugins that will be executed in a chain are for Pods that request this network are: _sriov_, _rdma_ CNIs

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: sriov-rdma-net
  annotations:
    k8s.v1.cni.cncf.io/resourceName: mellanox.com/sriov_rdma
spec:
  config: '{
             "cniVersion": "0.3.1",
             "name": "sriov-rdma-net",
             "plugins": [{
                          "type": "sriov",
                          "ipam": {
                            "type": "host-local",
                            "subnet": "10.56.217.0/24",
                            "routes": [{
                              "dst": "0.0.0.0/0"
                            }],
                            "gateway": "10.56.217.1"
                          }
                        },
                        {
                          "type": "rdma"
                        }]
           }'
```

# Development
It is recommended to use the same go version as defined in `.travis.yml`
to avoid potential build related issues during development (newer version will most likely work as well).

### Build from source
```console
~$ git clone https://github.com/Mellanox/rdma-cni.git
~$ cd rdma-cni
~$ make
```
Upon a successful build, `rdma` binary can be found under `./build`.
For small deployments (e.g a kubernetes test cluster/AIO K8s deployment) you can:
1. Copy `rdma` binary to the CNI dir in each worker node.
2. Build container image, push to your own image repo then modify the deployment template and deploy. 

#### Run tests:
```console
~$ make tests
```

#### Build image:
```console
~$ make image
```
