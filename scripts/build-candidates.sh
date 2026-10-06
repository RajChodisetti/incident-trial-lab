#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
test "$(go env GOVERSION)/$(go env GOARCH)" = go1.26.2/arm64 || { echo 'Requires Go 1.26.2 and arm64' >&2; exit 1; }
builder=incident-trial-lab-m3-builder
if docker buildx inspect "$builder" >/dev/null 2>&1; then echo 'Refusing existing project builder' >&2; exit 1; fi
docker buildx create --name "$builder" --driver docker-container --driver-opt image=moby/buildkit@sha256:cec9f139f45e93c5c69c60f8b07cfad9f43f4ef6b6a6cd917527fea5ff2e3dea
trap 'docker buildx rm "$builder"' EXIT
trap 'exit 130' INT TERM
for variant in fixed ineffective misleading; do
    dir=.build/candidates/$variant
    mkdir -p "$dir"
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false -ldflags="-buildid= -X main.variant=$variant" -o "$dir/synthetic-service" ./cmd/candidate-service
    go run ./cmd/build-binding "$variant" > "$dir/source-binding.json"
    source_sha=$(jq -er '.source_sha256' "$dir/source-binding.json")
    logic_sha=$(jq -er '.logic_sha256' "$dir/source-binding.json")
    binary_sha=$(jq -er '.binary_sha256' "$dir/source-binding.json")
    cp deploy/local/Dockerfile.candidate "$dir/Dockerfile"
    docker buildx build --builder "$builder" --build-arg SOURCE_DATE_EPOCH=0 --build-arg VARIANT="$variant" --build-arg SOURCE_SHA256="$source_sha" --build-arg LOGIC_SHA256="$logic_sha" --build-arg BINARY_SHA256="$binary_sha" --platform linux/arm64 --provenance=false --output type=oci,rewrite-timestamp=true,dest="$dir/image.oci.tar" --metadata-file "$dir/metadata.json" "$dir"
    if [ -f deploy/local/candidates.json ]; then
        expected=$(jq -er --arg variant "$variant" '.[$variant]' deploy/local/candidates.json)
        actual=$(jq -er '."containerimage.digest"' "$dir/metadata.json")
        test "$expected" = "$actual" || { echo "Digest changed for $variant; deliberate lock review required" >&2; exit 1; }
    fi
done
