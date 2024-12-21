FROM --platform=$BUILDPLATFORM golang:1.23 as builder
ARG TARGETOS TARGETARCH

WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH make

FROM scratch

COPY --from=builder /go/src/app/mikrotik-exporter /bin/mikrotik-exporter
ENTRYPOINT ["/bin/mikrotik-exporter"]
