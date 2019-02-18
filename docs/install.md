## Setup

0\. Prepare GPU Node

This Guide assumes that the NVIDIA drivers and nvidia-docker2 have been installed.

Enable the nvidia runtime as your default runtime on your node. To do this, please edit the docker daemon config file which is usually present at /etc/docker/daemon.json:

```
{
    "default-runtime": "nvidia",
    "runtimes": {
        "nvidia": {
            "path": "/usr/bin/nvidia-container-runtime",
            "runtimeArgs": []
        }
    }
}
```

> *if `runtimes` is not already present, head to the install page of [nvidia-docker](https://github.com/NVIDIA/nvidia-docker)*

1\. Deploy GPU share scheduler extender

```
cd /etc/kubernetes/
curl -O https://raw.githubusercontent.com/AliyunContainerService/gpushare-scheduler-extender/master/config/scheduler-policy-config.json
cd /tmp/
curl -O https://raw.githubusercontent.com/AliyunContainerService/gpushare-scheduler-extender/master/config/gpushare-schd-extender.yaml
kubectl create -f gpushare-schd-extender.yaml
```

2\. Modify scheduler configuration to add `/etc/kubernetes/scheduler-policy-config.json`, here is the sample of the modified [kube-scheduler.yaml](../config/kube-scheduler.yaml)

2.1 Add Policy config file parameter in scheduler arguments

```
- --policy-config-file=/etc/kubernetes/scheduler-policy-config.json
```

2.2 Add volume mount into Pod Spec

```
- mountPath: /etc/kubernetes/scheduler-policy-config.json
  name: scheduler-policy-config
  readOnly: true
```

```
- hostPath:
      path: /etc/kubernetes/scheduler-policy-config.json
      type: FileOrCreate
  name: scheduler-policy-config
```

> Notice: If your Kubernetes default scheduler is deployed as static pod, don't edit the yaml file inside /etc/kubernetes/manifest. You need to edit the yaml file outside the `/etc/kubernetes/manifest` directory.

3\. Deploy Device Plugin

```
wget https://raw.githubusercontent.com/AliyunContainerService/gpushare-device-plugin/master/device-plugin-rbac.yaml
kubectl create -f device-plugin-rbac.yaml
wget https://raw.githubusercontent.com/AliyunContainerService/gpushare-device-plugin/master/device-plugin-ds.yaml
kubectl create -f device-plugin-ds.yaml
```

4\. Add gpushare node labels to the nodes requiring GPU sharing

```
kubectl label node <target_node> gpushare=true
```

```
kubectl label no mynode gpushare=true
```

5\. Install Kubectl extension


5.1 Install kubectl 1.12 or above. You can download and install `kubectl` for linux

```
curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.12.1/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl
```

5.2 Download and install the kubectl extension

```
curl -o /usr/bin/kubectl-inspect-gpushare https://github.com/AliyunContainerService/gpushare-device-plugin/releases/download/v0.1.0/arenakubectl-inspect-gpushare-v2
chmod u+x /usr/bin/kubectl-inspect-gpushare
```
