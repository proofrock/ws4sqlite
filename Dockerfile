# docker build -t ws4sqlite .

FROM alpine:edge AS build

RUN apk update
RUN apk upgrade
RUN apk add --update go git make
WORKDIR /app
ENV GOPATH /app
RUN git clone https://github.com/proofrock/ws4sqlite
WORKDIR /app/ws4sqlite
RUN make build-nostatic

FROM alpine:latest

COPY --from=build /app/ws4sqlite/bin/ws4sqlite /

EXPOSE 12321
VOLUME /data

ENTRYPOINT ["/ws4sqlite"]
