FROM golang:1.17 AS builder

ARG GO_LDFLAGS
ARG TARGETARCH

WORKDIR /code
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -v -o /usr/local/bin/edgemesh-server -ldflags "$GO_LDFLAGS -w -s" \
./server/cmd/edgemesh-server/server.go


FROM alpine:3.11

COPY --from=builder /usr/local/bin/edgemesh-server /usr/local/bin/edgemesh-server

ENTRYPOINT ["edgemesh-server"]
