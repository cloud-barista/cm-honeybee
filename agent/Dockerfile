FROM alpine:3.19.1

RUN apk --no-cache add tzdata iptables podman
RUN echo "Asia/Seoul" >  /etc/timezone
RUN cp -f /usr/share/zoneinfo/Asia/Seoul /etc/localtime

RUN mkdir /conf
COPY cmd/cm-honeybee-agent/cm-honeybee-agent /cm-honeybee-agent
RUN chmod 755 /cm-honeybee-agent

USER root
ENTRYPOINT ["./docker-entrypoint.sh"]
