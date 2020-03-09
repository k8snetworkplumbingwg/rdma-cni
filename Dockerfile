FROM golang:alpine as builder

ADD . /usr/src/rdma-cni

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

RUN apk add --update --virtual build-dependencies build-base linux-headers && \
    cd /usr/src/rdma-cni && \
    make clean && \
    make build

FROM alpine
COPY --from=builder /usr/src/rdma-cni/build/rdma /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="RDMA CNI"

ADD ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
