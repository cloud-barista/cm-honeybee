FROM golang:1.21.6-bookworm AS builder

RUN apt-get update && apt-get install -y make bash git

WORKDIR /go/src/github.com/cloud-barista/cm-honeybee/server

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN git config --global user.email "ish@innogrid.com"
RUN git config --global user.name "ish-hcc"
RUN git init
RUN git commit --allow-empty -m "a commit for the build"

RUN make

FROM alpine:3.20.1 as prod

RUN apk --no-cache add tzdata
RUN echo "Asia/Seoul" >  /etc/timezone
RUN cp -f /usr/share/zoneinfo/Asia/Seoul /etc/localtime

COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/server/conf /conf
COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/server/cmd/cm-honeybee/cm-honeybee /cm-honeybee

RUN mkdir -p /root/.cm-honeybee/
COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/server/_default_key/honeybee.pub /root/.cm-honeybee/honeybee.pub
COPY --from=builder /go/src/github.com/cloud-barista/cm-honeybee/server/_default_key/honeybee.key /root/.cm-honeybee/honeybee.key

USER root
CMD ["/cm-honeybee"]

EXPOSE 8081
