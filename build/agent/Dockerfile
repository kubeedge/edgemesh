FROM golang:1.14-alpine3.11 AS builder

ARG GO_LDFLAGS

# set goproxy
# using the go module
RUN go env -w GO111MODULE=on
# source
RUN apk update && apk add gcc && \
    apk --no-cache add build-base linux-headers sqlite-dev binutils-gold &&\
    apk --no-cache add iptables

WORKDIR /code
COPY . .

RUN CGO_ENABLED=1 go build -v -o /usr/local/bin/edgemesh-agent -ldflags "$GO_LDFLAGS -w -s" \
./agent/cmd/edgemesh-agent/agent.go


FROM alpine:3.11

RUN apk update && apk --no-cache add iptables

COPY --from=builder /usr/local/bin/edgemesh-agent /usr/local/bin/edgemesh-agent

ENTRYPOINT ["edgemesh-agent"]
