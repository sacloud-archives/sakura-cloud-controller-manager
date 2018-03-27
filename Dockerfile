FROM golang:1.10 AS builder
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

FROM alpine:3.7
MAINTAINER Kazumichi Yamamoto <yamamoto.febc@gmail.com>
LABEL MAINTAINER 'Kazumichi Yamamoto <yamamoto.febc@gmail.com>'

RUN set -x && apk add --no-cache --update ca-certificates
COPY --from=builder /go/src/github.com/sacloud/sakura-cloud-controller-manager/bin/sakura-cloud-controller-manager /usr/local/bin/
RUN chmod +x /usr/local/bin/sakura-cloud-controller-manager
ENTRYPOINT ["/usr/local/bin/sakura-cloud-controller-manager"]
