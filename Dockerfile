# docker build -t ws4sqlite .
FROM alpine:latest

COPY bin/ws4sqlite /

EXPOSE 12321
VOLUME /data

ENTRYPOINT ["/ws4sqlite", "--cfgDir=/data"]
