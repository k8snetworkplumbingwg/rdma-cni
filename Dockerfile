FROM golang:alpine as builder

COPY . /usr/src/rdma-cni

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

WORKDIR /usr/src/rdma-cni
RUN apk add --no-cache --virtual build-dependencies build-base=~0.5 && \
    make clean && \
    make build

FROM alpine:3
COPY --from=builder /usr/src/rdma-cni/build/rdma /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="RDMA CNI"

COPY ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
