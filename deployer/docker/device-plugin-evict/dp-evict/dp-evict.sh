#!/usr/bin/env bash
set -xe

if [ -d "/k8s-host" ]; then
	rm -rf /k8s-host/usr/local/dp-dvict
	mkdir -p /k8s-host/usr/local/dp-dvict
	cp -r /schd-extender/* /k8s-host/usr/local/dp-dvict
	chmod -R +x /dp-evict/usr/local/dp-dvict/
	chroot /dp-evict /usr/local/dp-dvict/dp-evict-on-host.sh "$@"
fi