FROM golang:1.17 AS builder

ARG GO_LDFLAGS
ARG TARGETARCH

WORKDIR /code
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GO_LDFLAGS=$GO_LDFLAGS make WHAT=edgemesh-agent


FROM alpine:3.11

RUN apk update && apk --no-cache add iptables

COPY --from=builder /code/_output/local/bin/edgemesh-agent /usr/local/bin/edgemesh-agent

ENTRYPOINT ["edgemesh-agent"]
