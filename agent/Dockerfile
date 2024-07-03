FROM golang:1.21.6-bookworm AS builder

RUN apt-get update && apt-get install -y make bash git

WORKDIR /go/src/github.com/cloud-barista/cm-honeybee/agent

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN git config --global user.email "ish@innogrid.com"
RUN git config --global user.name "ish-hcc"
RUN git init
RUN git commit --allow-empty -m "a commit for the build"

RUN make

FROM alpine:3.20.1

RUN apk --no-cache add tzdata iptables podman
RUN echo "Asia/Seoul" >  /etc/timezone
RUN cp -f /usr/share/zoneinfo/Asia/Seoul /etc/localtime

COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/agent/conf /conf
COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/agent/cmd/cm-honeybee-agent/cm-honeybee-agent /cm-honeybee-agent
COPY docker-entrypoint.sh /docker-entrypoint.sh

USER root
ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 8082
