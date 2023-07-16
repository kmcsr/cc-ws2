#!/bin/bash

PUBLIC_PREFIX=craftmine/ccd
BUILD_PLATFORMS=(linux/arm64 linux/amd64) #

NPM_DIR=vue-project

cd $(dirname $0)

[ -n "$TAG" ] || TAG=$(git describe --tags --match v[0-9]* --abbrev=0 2>/dev/null || git log -1 --format="dev-%H")

function build(){
	tag=$1
	platform=$2
	fulltag="${PUBLIC_PREFIX}:${tag}"
	echo
	echo "==> building $fulltag from Dockerfile.$tag"
	echo
	DOCKER_BUILDKIT=1 docker build --platform ${platform} \
	 --tag "$fulltag" \
	 --file "Dockerfile.$tag" \
	 .. || return $?
	echo
	if [ -n "$TAG" ]; then
		docker tag "$fulltag" "${fulltag}-${TAG}" || return $?
		echo "==> pushing $fulltag ${fulltag}-${TAG}"
		echo
		(docker push "$fulltag" && docker push "${fulltag}-${TAG}") || return $?
	fi
	return 0
}

echo

for platform in "${BUILD_PLATFORMS[@]}"; do
	build web $platform || exit $?
done
