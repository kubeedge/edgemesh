FROM golang:1.17 AS builder

ARG GO_LDFLAGS
ARG TARGETARCH

WORKDIR /code
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -v -o /usr/local/bin/edgemesh-agent -ldflags "$GO_LDFLAGS -w -s" \
./agent/cmd/edgemesh-agent/agent.go


FROM alpine:3.11

RUN apk update && apk --no-cache add iptables

COPY --from=builder /usr/local/bin/edgemesh-agent /usr/local/bin/edgemesh-agent

ENTRYPOINT ["edgemesh-agent"]
