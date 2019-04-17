#!/usr/bin/env bash

set -e -x

backup_dir="/etc/kubernetes/manifests-backup"

backup_file="$backup_dir/kube-scheduler-$(date +%Y%m%d_%H%M%S).yaml"

public::common::log() {
	echo $(date +"[%Y%m%d %H:%M:%S]: ") $1
}

public::deployer::sche-policy-config() {
	dir=/etc/kubernetes/manifests/
	if ! grep policy-config-file $dir/kube-scheduler.yaml; then
		backup_dir="/etc/kubernetes/manifests-backup/$KUBE_VERSION"
		mkdir -p $backup_dir
		cp /etc/kubernetes/manifests/kube-scheduler.yaml ${backup_file}
		public::common::log "Backup $backup_dir/kube-scheduler-$(date +%Y%m%d_%H%M%S).yaml"
		public::common::log "Configure shceduler extender"
		cp /usr/local/k8s-schd-extender/scheduler-policy-config.json /etc/kubernetes/scheduler-policy-config.json
		if ! grep 'policy-config-file=/etc/kubernetes/scheduler-policy-config.json' $dir/kube-scheduler.yaml; then
			sed -i "/- kube-scheduler/a\ \ \ \ - --policy-config-file=/etc/kubernetes/scheduler-policy-config.json" $dir/kube-scheduler.yaml
		fi
		# add scheduler config policy volumeMounts
		if ! grep 'mountPath: /etc/kubernetes/scheduler-policy-config.json' $dir/kube-scheduler.yaml; then
			sed -i "/  volumeMounts:/a\ \ \ \ - mountPath: /etc/kubernetes/scheduler-policy-config.json\n      name: scheduler-policy-config\n      readOnly: true" $dir/kube-scheduler.yaml
		fi
		# add scheduler config policy volumes
		if ! grep 'path: /etc/kubernetes/scheduler-policy-config.json' $dir/kube-scheduler.yaml; then
			sed -i "/  volumes:/a \  - hostPath:\n      path: /etc/kubernetes/scheduler-policy-config.json\n      type: FileOrCreate\n    name: scheduler-policy-config" $dir/kube-scheduler.yaml
		fi
	else
		public::common::log "Skip the kube-scheduler config, because it's already configured extender."
	fi
}

main() {
	while
		[[ $# -gt 0 ]]
	do
		key="$1"

		case $key in
		--version)
			KUBE_VERSION=$2
			export KUBE_VERSION=${KUBE_VERSION:1}
			shift
			;;
		*)
			public::common::log "unkonw option [$key]"
			exit 1
			;;
		esac
		shift
	done

	if [ "$KUBE_VERSION" == "" ]; then
		# Using default cidr.
		public::common::log "KUBE_VERSION $KUBE_VERSION is not set."
		exit 1
	fi

	public::deployer::sche-policy-config

	touch /ready
}

main "$@"