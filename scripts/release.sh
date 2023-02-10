#! /bin/bash

version=$1
rm -r ./release 2> /dev/null
mkdir release

function build() {
    echo "building binary for ${GOOS}/${GOARCH}..."
    go build -ldflags "-X main.versionNumber=${version}" -o "./release/klient.${GOOS}.${GOARCH}" && echo "successfully built klient.${GOOS}.${GOARCH}" || echo "error building klient.${GOOS}.${GOARCH}"
}

export CGO_ENABLED=0

export GOOS=linux
export GOARCH=amd64
build

export GOOS=darwin
export GOARCH=amd64
build

export GOOS=darwin
export GOARCH=arm64
build

echo "release builds completed"