#!/bin/bash

for f in $(find . -name "*.proto"); do
    protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
           "$f"
done
for f in `find . -name "*_http.yaml"`
do
    protoc -I . --grpc-gateway_out=logtostderr=true,paths=source_relative,grpc_api_configuration=$f:. ${f/_http.yaml/.proto}
done
