#!/bin/bash
# This script regenerates the Go gRPC code from the .proto file.
# Requires protoc and protoc-gen-go plugins to be installed.
echo "Generating gRPC code..."
protoc --go_out=. --go_opt=paths=source_relative 
       --go-grpc_out=. --go-grpc_opt=paths=source_relative 
       api/v1/synapse.proto
