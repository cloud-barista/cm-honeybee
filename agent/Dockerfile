# Run the toolchain on the native build platform and cross-compile to the
# target platform so multi-arch builds (e.g. linux/arm64) don't emulate the
# whole Go compile under QEMU.
FROM --platform=$BUILDPLATFORM golang:1.26.2-bookworm AS builder

# Populated automatically by BuildKit (e.g. TARGETARCH=arm64). Empty when built
# with the legacy builder, in which case the build falls back to the host arch.
ARG TARGETOS
ARG TARGETARCH

RUN apt-get update && apt-get install -y make bash git

WORKDIR /go/src/github.com/cloud-barista/cm-honeybee/agent

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN git config --global user.email "ish@innogrid.com"
RUN git config --global user.name "ish-hcc"
RUN git init
RUN git commit --allow-empty -m "a commit for the build"

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build-only

FROM alpine:3.20.1 as prod

RUN apk --no-cache add tzdata iptables nftables podman
RUN echo "Asia/Seoul" >  /etc/timezone
RUN cp -f /usr/share/zoneinfo/Asia/Seoul /etc/localtime

COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/agent/conf /conf
COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/agent/cmd/cm-honeybee-agent/cm-honeybee-agent /cm-honeybee-agent
COPY docker-entrypoint.sh /docker-entrypoint.sh

USER root
ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 8082
