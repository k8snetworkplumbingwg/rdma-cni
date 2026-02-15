FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS

COPY . /usr/src/rdma-cni
WORKDIR /usr/src/rdma-cni

ENV HTTP_PROXY=$http_proxy
ENV HTTPS_PROXY=$https_proxy

RUN apk add --no-cache build-base=~0.5 \
    && make clean && make build TARGET_OS=$TARGETOS TARGET_ARCH=$TARGETARCH

FROM alpine:3
COPY --from=builder /usr/src/rdma-cni/build/rdma /usr/bin/
COPY ./images/entrypoint.sh /

WORKDIR /
LABEL io.k8s.display-name="RDMA CNI"
ENTRYPOINT ["/entrypoint.sh"]
