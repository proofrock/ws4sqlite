FROM gcr.io/distroless/static-debian12

COPY bin/ws4sql /

EXPOSE 12321
VOLUME /data

ENTRYPOINT ["/ws4sql"]