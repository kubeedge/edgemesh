FROM golang:1.19 AS builder

ARG GO_LDFLAGS
ARG TARGETARCH

WORKDIR /code
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GO_LDFLAGS=$GO_LDFLAGS make WHAT=edgemesh-cni


FROM alpine:3.11

RUN apk update && apk --no-cache add iptables && apk --no-cache add dpkg

COPY --from=builder /code/_output/local/bin/edgemesh-cni /usr/local/bin/edgemesh-cni

# Copy cni cfg
COPY --from=builder /code/build/cni/init-cni /usr/local/init-cni

ENTRYPOINT ["sh", "-c", "/usr/local/init-cni && echo 'CNI pre-install completed, exiting...' && exit 0"]