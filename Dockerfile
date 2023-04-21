# docker build -t ws4sqlite .

FROM golang:latest as build

WORKDIR /go/src/app
COPY . .

# RUN CGO_ENABLED=0 go build -o /go/bin/ws4sqlite -trimpath
RUN make build-nostatic

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian11
COPY --from=build /go/src/app/bin/ws4sqlite /

ENTRYPOINT ["/ws4sqlite"]