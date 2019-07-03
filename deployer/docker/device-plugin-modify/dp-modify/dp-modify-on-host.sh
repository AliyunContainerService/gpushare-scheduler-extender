#!/usr/bin/env bash

set -e -x

backup_dir="/etc/kubernetes/manifests-backup"

public::common::log() {
	echo $(date +"[%Y%m%d %H:%M:%S]: ") $1
}

public::modify::gpu-device-plugin() {
	dir=/k8s-host/etc/kubernetes/manifests/

	if [  -f /k8s-host/etc/kubernetes/manifests/nvidia-device-plugin.yml ]; then

			python /dp-modify/dp-modify.py --requestcpu=${REQUEST_CPU} --requestmemory=${REQUEST_MEMORY} \
								--limitcpu=${LIMIT_CPU} --limitmemory=${LIMIT_MEMORY} \
								--image=${IMAGE}

		else
			public::common::log "Skip removing nvidia-device-plugin.yml, because it doesn't exist."

	fi
}

public::common::parse-args() {
    while [[ $# -gt 1 ]]
    do
    key="$1"

    case $key in
        --image)
            export IMAGE=$2
            shift
        ;;
        --request-cpu)
            export REQUEST_CPU=$2
            shift
        ;;
        --request-memory)
            export REQUEST_MEMORY=$2
            shift
        ;;
        --limit-cpu)
            export LIMIT_CPU=$2
            shift
        ;;
        --limit-memory)
            export LIMIT_MEMORY=$2
            shift
        ;;
        *)
            # unknown option
            public::common::log "unkonw option [$key]"
        ;;
    esac
    shift
    done

}
main() {
	public::common::parse-args "$@"

	public::modify::gpu-device-plugin

	touch /ready
}

main "$@"
