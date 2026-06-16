FROM golang:1.26 as builder
RUN go install github.com/golang/dep/cmd/dep@latest
WORKDIR /go/src/github.com/influxdata/influxdb
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure -vendor-only
COPY . /go/src/github.com/influxdata/influxdb
RUN go install ./cmd/...

FROM debian:stretch
COPY --from=builder /go/bin/* /usr/bin/
COPY --from=builder /go/src/github.com/influxdata/influxdb/etc/config.sample.toml /etc/influxdb/influxdb.conf

EXPOSE 8086
EXPOSE 6060
VOLUME /var/lib/influxdb

COPY docker/entrypoint.sh /entrypoint.sh
COPY docker/init-influxdb.sh /init-influxdb.sh
ENTRYPOINT ["/entrypoint.sh"]
CMD ["influxd"]
