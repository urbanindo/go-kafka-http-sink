#!/bin/bash

if ! command -v swag &>/dev/null; then
  echo "swag not found. Attemp install swag binary"
  go install github.com/swaggo/swag/cmd/swag@latest >/dev/null
fi

echo "formatting open API annotation"
swag fmt -d internal

echo "generating open API specs"
swag init -g server.go -o ./api --pd -d internal/handler
