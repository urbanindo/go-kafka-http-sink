
################################################################################
# Docker file for building image
################################################################################

FROM golang:1.21 AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN go env -w GOPRIVATE=github.com/urbanindo/*

ARG NETRC
RUN echo ${NETRC} > ~/.netrc && chmod 600 ~/.netrc

WORKDIR /go/src/go-kafka-http-sink
COPY . /go/src/go-kafka-http-sink
RUN go mod download
RUN go mod verify

ARG CMD_PATH

RUN go build -v -o output/app ./${CMD_PATH}/...

# Final Image
################################################################################

FROM alpine:latest

ARG CMD_PATH
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/
COPY --from=builder /go/src/go-kafka-http-sink/output/ .

CMD ["./app"]
