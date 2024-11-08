FROM golang:1.21.0-alpine
RUN apk --no-cache add gcc g++ make ca-certificates curl git openssh tzdata
ENV TZ=Asia/Jakarta \
    GOPRIVATE=github.com/urbanindo/*

WORKDIR /app

RUN go env -w GOPRIVATE=$GOPRIVATE && \
    echo $GOPRIVATE

ADD build/ci/.netrc /root/.netrc

ARG CMD_PATH
ENV CMD_PATH=${CMD_PATH}

RUN go install github.com/githubnemo/CompileDaemon@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest

COPY go.mod go.sum ./

RUN go mod download
RUN go mod verify

WORKDIR /go/src/go-kafka-http-sink

ENTRYPOINT CompileDaemon -exclude-dir=".git" -build="go build -v -o /go/bin/app /go/src/go-kafka-http-sink/${CMD_PATH}/main.go" -command="/go/bin/app"
