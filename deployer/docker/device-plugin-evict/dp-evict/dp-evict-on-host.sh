#!/usr/bin/env bash

set -e -x

backup_dir="/etc/kubernetes/manifests-backup"

public::common::log() {
	echo $(date +"[%Y%m%d %H:%M:%S]: ") $1
}

public::deployer::sche-policy-config() {
	dir=/etc/kubernetes/manifests/

	if [  -f /etc/kubernetes/manifests/nvidia-device-plugin.yml ]; then

		else
			public::common::log "Skip removing, because it's already configured extender."
	fi
	if ! grep policy-config-file $dir/kube-scheduler.yaml; then
		backup_dir="/etc/kubernetes/manifests-backup/$KUBE_VERSION"
		mkdir -p $backup_dir
		cp /etc/kubernetes/manifests/kube-scheduler.yaml $backup_dir/kube-scheduler-$(date +%Y%m%d_%H%M%S).yaml
			else
		public::common::log "Skip the kube-scheduler config, because it's already configured extender."
	fi
}

main() {

	if [ "$KUBE_VERSION" == "" ]; then
		# Using default cidr.
		public::common::log "KUBE_VERSION $KUBE_VERSION is not set."
		exit 1
	fi

	public::deployer::sche-policy-config

	touch /ready
	sleep infinity
}