#!/usr/bin/env bash
set -xe

if [ -d "/k8s-host" ]; then
	bash /dp-modify/dp-modify-on-host.sh "$@"
	while sleep 3600; do :; done
fi
