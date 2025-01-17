# See BUILDING.md

FROM golang:latest AS build

WORKDIR /go/src/app
COPY . .

RUN make build-static

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian12
COPY --from=build /go/src/app/bin/ws4sql /

EXPOSE 12321
VOLUME /data

ENTRYPOINT ["/ws4sql"]