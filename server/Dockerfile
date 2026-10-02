# Run the toolchain on the native build platform and cross-compile to the
# target platform so multi-arch builds (e.g. linux/arm64) don't emulate the
# whole Go compile under QEMU.
FROM --platform=$BUILDPLATFORM golang:1.27.1-bookworm AS builder

# Populated automatically by BuildKit (e.g. TARGETARCH=arm64). Empty when built
# with the legacy builder, in which case the build falls back to the host arch.
ARG TARGETOS
ARG TARGETARCH

RUN apt-get update && apt-get install -y make bash git

WORKDIR /go/src/github.com/cloud-barista/cm-honeybee/server

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# cb-spider drivers read their CSP metainfo and log configs from
# $CBSPIDER_ROOT and $CBLOG_ROOT at runtime. Take them from the cb-spider version pinned in go.mod.
RUN SPIDER_DIR=`go list -m -f '{{.Dir}}' github.com/cloud-barista/cb-spider` && \
    mkdir -p /cb-spider/cloud-driver-libs/region /cb-spider/conf && \
    cp $SPIDER_DIR/cloud-driver-libs/cloudos.yaml $SPIDER_DIR/cloud-driver-libs/cloudos_meta.yaml /cb-spider/cloud-driver-libs/ && \
    cp $SPIDER_DIR/cloud-driver-libs/region/*_region_meta.yaml /cb-spider/cloud-driver-libs/region/ && \
    cp $SPIDER_DIR/conf/calllog_conf.yaml $SPIDER_DIR/conf/log_conf.yaml /cb-spider/conf/ && \
    chmod -R u+w /cb-spider

COPY . .

RUN git config --global user.email "ish@innogrid.com"
RUN git config --global user.name "ish-hcc"
RUN git init
RUN git commit --allow-empty -m "a commit for the build"

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build-only

FROM alpine:3.24.2 as prod

RUN apk --no-cache add tzdata
RUN echo "Asia/Seoul" >  /etc/timezone
RUN cp -f /usr/share/zoneinfo/Asia/Seoul /etc/localtime

COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/server/conf /conf
COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/server/cmd/cm-honeybee/cm-honeybee /cm-honeybee

# Kept outside /root/.cm-honeybee, which deployments bind-mount from the host.
COPY --from=builder /cb-spider /cb-spider
ENV CBSPIDER_ROOT=/cb-spider
ENV CBLOG_ROOT=/cb-spider

RUN mkdir -p /root/.cm-honeybee/
COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/server/_default_key/honeybee.pub /root/.cm-honeybee/honeybee.pub
COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/server/_default_key/honeybee.key /root/.cm-honeybee/honeybee.key

USER root
CMD ["/cm-honeybee"]

EXPOSE 8081
