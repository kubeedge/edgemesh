FROM golang:1.17 AS builder

ARG GO_LDFLAGS
ARG TARGETARCH

WORKDIR /code
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GO_LDFLAGS=$GO_LDFLAGS make WHAT=edgemesh-agent


FROM alpine:3.11

RUN apk update && apk --no-cache add iptables && apk --no-cache add dpkg

COPY --from=builder /code/_output/local/bin/edgemesh-agent /usr/local/bin/edgemesh-agent

# Add support for auto-detection of iptables mode
COPY --from=builder /code/build/agent/iptables-wrapper /sbin/iptables-wrapper
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-legacy 50
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-nft 50
RUN update-alternatives --install /sbin/iptables iptables /sbin/iptables-wrapper 100 \
    --slave /sbin/iptables-restore iptables-restore /sbin/iptables-wrapper \
    --slave /sbin/iptables-save iptables-save /sbin/iptables-wrapper

ENTRYPOINT ["edgemesh-agent"]
