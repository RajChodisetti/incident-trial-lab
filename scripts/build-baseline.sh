#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
test "$(go env GOVERSION)" = go1.26.2 || { echo 'Baseline build requires Go 1.26.2' >&2; exit 1; }
test "$(go env GOARCH)" = arm64 || { echo 'This initial local baseline targets arm64' >&2; exit 1; }
mkdir -p .build/image
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false -ldflags='-buildid=' -o .build/image/synthetic-service ./cmd/synthetic-service
cp deploy/local/Dockerfile .build/image/Dockerfile
builder=incident-trial-lab-m2-builder
if docker buildx inspect "$builder" >/dev/null 2>&1; then
    echo "Refusing to reuse existing builder $builder" >&2; exit 1
fi
docker buildx create --name "$builder" --driver docker-container --driver-opt image=moby/buildkit@sha256:cec9f139f45e93c5c69c60f8b07cfad9f43f4ef6b6a6cd917527fea5ff2e3dea
trap 'docker buildx rm "$builder"' EXIT
# Normalize image timestamps for reproducible artifacts; commits retain real dates.
docker buildx build --builder "$builder" --build-arg SOURCE_DATE_EPOCH=0 --platform linux/arm64 --provenance=false --output type=oci,rewrite-timestamp=true,dest=.build/baseline.oci.tar --metadata-file .build/image-metadata.json .build/image
