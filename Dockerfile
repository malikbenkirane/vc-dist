FROM golang:1.20-alpine AS builder

RUN apk add gzip tar

COPY . /build_dir

WORKDIR /build_dir

ENV GOOS darwin
ENV GOARCH arm64

RUN \
go build ./cmd/vc && \
tar -cf vc_darwin_arm64.tar vc && \
gzip vc_darwin_arm64.tar

FROM nginx AS server

COPY \
--from=builder \
/build_dir/vc_darwin_arm64.tar.gz \
/usr/share/nginx/html/vc_darwin_arm64.tar.gz