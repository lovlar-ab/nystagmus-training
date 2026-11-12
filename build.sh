#!/usr/bin/env bash
# Builds dist/NystagmusTraining.zip on Linux or macOS.
set -euo pipefail
cd "$(dirname "$0")"

app="dist/Nystagmus Training.app"
tmp=.tmp/build
rm -rf dist "$tmp"
mkdir -p "$app/Contents/MacOS" "$tmp"

go test ./internal/...
for arch in arm64 amd64; do
	GOOS=darwin GOARCH=$arch CGO_ENABLED=0 go vet .
	# The Go linker ad-hoc signs each slice.
	GOOS=darwin GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$tmp/$arch" .
done
go run ./tools/fat "$app/Contents/MacOS/nystagmus-training" "$tmp/arm64" "$tmp/amd64"
cp packaging/Info.plist "$app/Contents/Info.plist"

(cd dist && zip -qry NystagmusTraining.zip "Nystagmus Training.app")
echo "Built dist/NystagmusTraining.zip"
