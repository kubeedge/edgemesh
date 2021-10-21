FROM golang:1.14-alpine3.11 AS builder

ARG GO_LDFLAGS
ARG TARGETARCH

RUN go env -w GO111MODULE=on

WORKDIR /code
COPY . .

RUN GOOS=linux GOARCH=$TARGETARCH go build -v -o /usr/local/bin/edgemesh-server -ldflags "$GO_LDFLAGS -w -s" \
./server/cmd/edgemesh-server/server.go


FROM alpine:3.11

COPY --from=builder /usr/local/bin/edgemesh-server /usr/local/bin/edgemesh-server

ENTRYPOINT ["edgemesh-server"]
