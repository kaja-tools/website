#!/bin/sh

protoc --go_out=../internal/api --go_opt=paths=source_relative \
    --go-grpc_out=require_unimplemented_servers=false:../internal/api --go-grpc_opt=paths=source_relative \
    teams.proto 