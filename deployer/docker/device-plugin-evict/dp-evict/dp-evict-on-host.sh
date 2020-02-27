#!/usr/bin/env bash

set -e -x

backup_dir="/etc/kubernetes/manifests-backup"

public::common::log() {
	echo $(date +"[%Y%m%d %H:%M:%S]: ") $1
}

public::evict::gpu-device-plugin() {
	dir=/etc/kubernetes/manifests/

	if [  -f /etc/kubernetes/manifests/nvidia-device-plugin.yml ]; then
			backup_dir="/etc/kubernetes/manifests-backup/"
			mkdir -p $backup_dir
			mv /etc/kubernetes/manifests/nvidia-device-plugin.yml $backup_dir
		else
			public::common::log "Skip removing nvidia-device-plugin.yml, because it doesn't exist."
	fi
}

main() {

	public::evict::gpu-device-plugin

	touch /ready
	while sleep 3600; do :; done
}

main "$@"
