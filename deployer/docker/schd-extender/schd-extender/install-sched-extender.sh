#!/usr/bin/env bash
set -xe

if [ -d "/k8s-host" ]; then
	rm -rf /k8s-host/usr/local/k8s-schd-extender
	mkdir -p /k8s-host/usr/local/k8s-schd-extender
	cp -r /schd-extender/* /k8s-host/usr/local/k8s-schd-extender
	chmod -R +x /k8s-host/usr/local/k8s-schd-extender/
	chroot /k8s-host /usr/local/k8s-schd-extender/install-sched-extender-on-host.sh "$@"
	python3 /schd-extender/run.py
fi
