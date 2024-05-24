FROM alpine:3.19.1

RUN apk --no-cache add tzdata
RUN echo "Asia/Seoul" >  /etc/timezone
RUN cp -f /usr/share/zoneinfo/Asia/Seoul /etc/localtime

RUN mkdir /conf
COPY cmd/cm-honeybee/cm-honeybee /cm-honeybee
RUN chmod 755 /cm-honeybee

USER root
ENTRYPOINT ["./docker-entrypoint.sh"]
