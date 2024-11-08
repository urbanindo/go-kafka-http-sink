FROM golang:1.20.0-alpine
ARG go_version=1.20

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GO_VERSION=${go_version} \
    GOPRIVATE=github.com/urbanindo/*

RUN go env -w GOPRIVATE=$GOPRIVATE && \
    echo $GOPRIVATE

ADD build/ci/.netrc /root/.netrc

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download
RUN go mod verify