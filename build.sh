#!/bin/sh
#

set -e
set -f

###########################################

export CGO_ENABLED=0
export GO111MODULE=on
export GOPROXY=${GOPROXY:-https://mirrors.tencent.com/go/,https://proxy.golang.org,direct}

UPDATE_REPO=${UPDATE_REPO:-}
UPDATE_TOKEN=${UPDATE_TOKEN:-}

build() {
    echo building for $1/$2
    target=build/web-modem-$1-$2
    if [ x"$1" = x"windows" ]; then
        target="${target}.exe"
    fi

    ldflags="-s -w -X github.com/rehiy/web-modem/appinfo.Version=$last_tag"

    if [ -n "$UPDATE_REPO" ]; then
        ldflags="$ldflags -X github.com/rehiy/web-modem/appinfo.UpdateRepo=$UPDATE_REPO"
    fi
    if [ -n "$UPDATE_TOKEN" ]; then
        ldflags="$ldflags -X github.com/rehiy/web-modem/appinfo.UpdateToken=$UPDATE_TOKEN"
    fi

    GOOS=$1 GOARCH=$2 go build -ldflags="$ldflags" -o $target main.go
}

####################################################################

RUN_NUMBER=${GITHUB_RUN_NUMBER:-0}

last_tag=`git tag | sort -V | tail -n 1`
prev_tag=`git tag | sort -V | tail -n 2 | head -n 1`
git log $prev_tag..$last_tag --pretty=format:"%s" | grep -v "^release" | sed 's/^/- /' | sort > RELEASE.md

echo "build info - tag: $last_tag, build: $RUN_NUMBER"

####################################################################

build linux amd64
build linux arm64
build linux ppc64le
build linux s390x

build darwin amd64
build darwin arm64

build windows amd64
build windows arm64

####################################################################

for app in `ls build`; do
    gzip build/$app
done
