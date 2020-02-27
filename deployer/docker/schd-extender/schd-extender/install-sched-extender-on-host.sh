#!/usr/bin/env bash

set -e -x

dir=/etc/kubernetes/manifests

backup_dir="/etc/kubernetes/manifests-backup"

TIMESTAMP=$(date +%Y%m%d%H%M%S)

public::common::log() {
	echo $(date +"[%Y%m%d %H:%M:%S]: ") $1
}

public::deployer::sche-policy-config() {
	
	mkdir -p $backup_dir

    if [ ! -f $backup_dir/kube-scheduler.ori.yaml ];then
        cp /etc/kubernetes/manifests/kube-scheduler.yaml $backup_dir/kube-scheduler.ori.yaml
	    public::common::log "Backup $backup_dir/kube-scheduler.ori.yaml"
    else
	    cp /etc/kubernetes/manifests/kube-scheduler.yaml $backup_dir/kube-scheduler-$TIMESTAMP.yaml
	    public::common::log "Backup $backup_dir/kube-scheduler-$TIMESTAMP.yaml"
    fi

    if [ ! -f $backup_dir/scheduler-policy-config.ori.json ];then
        if [ -f /etc/kubernetes/scheduler-policy-config.json ];then
            cp /etc/kubernetes/scheduler-policy-config.json $backup_dir/scheduler-policy-config.ori.json
            public::common::log "Backup $backup_dir/scheduler-policy-config.ori.json"
        fi
    else
        if [ -f /etc/kubernetes/scheduler-policy-config.json ];then
            cp /etc/kubernetes/scheduler-policy-config.json $backup_dir/scheduler-policy-config-$TIMESTAMP.json
            public::common::log "Backup $backup_dir/scheduler-policy-config-$TIMESTAMP.json"
        fi
    fi

	public::common::log "Configure shceduler extender"
	cp -f /schd-extender/scheduler-policy-config.json /etc/kubernetes/scheduler-policy-config.json
    sed -i 's/127.0.0.1/'"${NODE_IP}"'/g' /etc/kubernetes/scheduler-policy-config.json
    if ! grep 'deployment.kubernetes.io/revision' $dir/kube-scheduler.yaml; then
        sed -i '/scheduler.alpha.kubernetes.io\/critical-pod/a \    deployment.kubernetes.io/revision: "'"${TIMESTAMP}"'"' $dir/kube-scheduler.yaml
    else
        # sed -i '/deployment.kubernetes.io\/revision/d' $dir/kube-scheduler.yaml
        sed -i 's#deployment.kubernetes.io/revision:.*#deployment.kubernetes.io/revision: "'"${TIMESTAMP}"'"#' $dir/kube-scheduler.yaml
    fi
    
	if ! grep 'policy-config-file=/etc/kubernetes/scheduler-policy-config.json' $dir/kube-scheduler.yaml; then
		sed -i "/- kube-scheduler/a\ \ \ \ - --policy-config-file=/etc/kubernetes/scheduler-policy-config.json" $dir/kube-scheduler.yaml
	else
		public::common::log "Skip the kube-scheduler config, because it's already configured extender."
	fi
	# add scheduler config policy volumeMounts
	if ! grep 'mountPath: /etc/kubernetes/scheduler-policy-config.json' $dir/kube-scheduler.yaml; then
		sed -i "/  volumeMounts:/a\ \ \ \ - mountPath: /etc/kubernetes/scheduler-policy-config.json\n      name: scheduler-policy-config\n      readOnly: true" $dir/kube-scheduler.yaml
	else
		public::common::log "Skip the scheduler-policy-config mountPath, because it's already configured extender."
	fi
	# add scheduler config policy volumes
	if ! grep 'path: /etc/kubernetes/scheduler-policy-config.json' $dir/kube-scheduler.yaml; then
		sed -i "/  volumes:/a \  - hostPath:\n      path: /etc/kubernetes/scheduler-policy-config.json\n      type: FileOrCreate\n    name: scheduler-policy-config" $dir/kube-scheduler.yaml
	else
		public::common::log "Skip the scheduler-policy-config volumes, because it's already configured extender."
	fi
}

main() {
	public::deployer::sche-policy-config

	touch /ready
	while sleep 3600; do :; done
}

main
