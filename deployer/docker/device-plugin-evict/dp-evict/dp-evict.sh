#!/usr/bin/env bash
set -xe

if [ -d "/k8s-host" ]; then
	rm -rf /k8s-host/usr/local/k8s-host
	mkdir -p /k8s-host/usr/local/k8s-host
	cp -r /schd-extender/* /k8s-host/usr/local/k8s-host
	chmod -R +x /dp-evict/usr/local/k8s-host/
	chroot /dp-evict /usr/local/k8s-host/dp-evict-on-host.sh "$@"
fi