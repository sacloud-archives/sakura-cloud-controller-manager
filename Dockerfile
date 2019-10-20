FROM golang:1.13 AS builder
MAINTAINER Kazumichi Yamamoto <yamamoto.febc@gmail.com>
LABEL MAINTAINER 'Kazumichi Yamamoto <yamamoto.febc@gmail.com>'

RUN  apt-get update && apt-get -y install \
        bash \
        git  \
        make \
        zip  \
      && apt-get clean \
      && rm -rf /var/cache/apt/archives/* /var/lib/apt/lists/*

ADD . /go/src/github.com/sacloud/sakura-cloud-controller-manager
WORKDIR /go/src/github.com/sacloud/sakura-cloud-controller-manager
RUN ["make","clean","build"]

#----------

FROM gcr.io/distroless/static:latest
LABEL MAINTAINER 'Kazumichi Yamamoto <yamamoto.febc@gmail.com>'
WORKDIR /
COPY --from=builder /go/src/github.com/sacloud/sakura-cloud-controller-manager/bin/sakura-cloud-controller-manager .
USER nobody
ENTRYPOINT ["/sakura-cloud-controller-manager"]