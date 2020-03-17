## Dockerfile build

This is used for distribution of RDMA CNI binary in a Docker image.

Typically you'd build this from the root of your RDMA CNI clone, and you'd set the `DOCKERFILE` to specify the Dockerfile during build time, `TAG` to specify the image's tag:

```
$ DOCKERFILE=Dockerfile TAG=mellanox/rdma-cni make image
```

---

## Daemonset deployment

You may wish to deploy RDMA CNI as a daemonset, you can do so by starting with the example Daemonset shown here:

```
$ kubectl create -f ../deployment/rdma-cni-daemonset.yaml
```

> __*Note:*__ The likely best practice here is to build your own image given the Dockerfile, and then push it to your preferred registry, and change the `image` fields in the Daemonset YAML to reference that image.

---

### Development notes

Example docker run command:

```
$ docker run -it -v /opt/cni/bin/:/host/opt/cni/bin/ --entrypoint=/bin/sh mellanox/rdma-cni
```

> __*Note:*__ `/opt/cni/bin` is assumed to be the CNI directory where CNI compliant executables are located.
