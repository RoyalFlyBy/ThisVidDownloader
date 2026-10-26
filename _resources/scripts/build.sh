#!/usr/bin/env sh
# Cross-compile into _bin/. Run from the repo root.
set -e
NAME=thisvid_downloader
for target in windows/amd64 windows/386 linux/amd64 linux/386 linux/arm linux/arm64 darwin/amd64 darwin/arm64 freebsd/amd64; do
	GOOS=${target%/*} GOARCH=${target#*/}
	EXT=""; [ "$GOOS" = "windows" ] && EXT=".exe"
	echo "building $GOOS/$GOARCH"
	CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags "-s -w" -o "_bin/${NAME}_${GOOS}_${GOARCH}${EXT}" ./_bin
done
