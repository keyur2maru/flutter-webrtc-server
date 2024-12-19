#!/bin/bash

# Create proto directories
mkdir -p proto/google/cloud/texttospeech/v1beta1
mkdir -p proto/google/api

# Set the source root directory for proto files
PROTO_SRC_ROOT=proto

# Generate Go code
protoc \
    -I $PROTO_SRC_ROOT \
    -I $PROTO_SRC_ROOT/google \
    -I $PROTO_SRC_ROOT/google/api \
    --go_out=. \
    --go_opt=paths=source_relative \
    --go-grpc_out=. \
    --go-grpc_opt=paths=source_relative \
    $PROTO_SRC_ROOT/google/api/client.proto \
    $PROTO_SRC_ROOT/google/api/launch_stage.proto \
    $PROTO_SRC_ROOT/google/cloud/texttospeech/v1beta1/texttospeech.proto \
    $PROTO_SRC_ROOT/google/api/http.proto \
    $PROTO_SRC_ROOT/google/api/annotations.proto \
    $PROTO_SRC_ROOT/google/api/field_behavior.proto \
    $PROTO_SRC_ROOT/google/api/resource.proto
