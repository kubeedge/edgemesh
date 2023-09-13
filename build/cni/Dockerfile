FROM golang:1.17 AS builder

ARG GO_LDFLAGS
ARG TARGETARCH

WORKDIR /code
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GO_LDFLAGS=$GO_LDFLAGS make WHAT=edgemesh-cni


FROM alpine:3.11

COPY --from=builder /code/_output/local/bin/edgemesh-cni /usr/local/bin/edgemesh-cni

ENTRYPOINT ["edgemesh-cni"]