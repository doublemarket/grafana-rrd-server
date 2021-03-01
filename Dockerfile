FROM golang:1.16-alpine AS builder

RUN apk update && apk add pkgconfig rrdtool-dev gcc libc-dev

WORKDIR /build
COPY . .
RUN go build -o grafana-rrd-server

FROM alpine
RUN apk add rrdtool rrdtool-dev
COPY --from=builder /build/grafana-rrd-server /grafana-rrd-server
ENTRYPOINT [ "/grafana-rrd-server" ]