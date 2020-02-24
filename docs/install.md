# Installation guide

## 0\. Prepare GPU Node

This guide assumes that the NVIDIA drivers and nvidia-docker2 have been installed.

Enable the Nvidia runtime as your default runtime on your node. To do this, please edit the docker daemon config file which is usually present at /etc/docker/daemon.json:

```json
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

## 1\. Deploy GPU share scheduler extender

```bash
cd /etc/kubernetes/
curl -O https://raw.githubusercontent.com/AliyunContainerService/gpushare-scheduler-extender/master/config/scheduler-policy-config.json
cd /tmp/
curl -O https://raw.githubusercontent.com/AliyunContainerService/gpushare-scheduler-extender/master/config/gpushare-schd-extender.yaml
kubectl create -f gpushare-schd-extender.yaml
```

## 2\. Modify scheduler configuration
The goal is to include `/etc/kubernetes/scheduler-policy-config.json` into the scheduler configuration (`/etc/kubernetes/manifests/kube-scheduler.yaml`).
Here is the sample of the modified [kube-scheduler.yaml](../config/kube-scheduler.yaml)

> Notice: If your Kubernetes default scheduler is deployed as static pod, don't edit the yaml file inside /etc/kubernetes/manifest. You need to edit the yaml file outside the `/etc/kubernetes/manifest` directory. and copy the yaml file you edited to the '/etc/kubernetes/manifest/' directory, and then kubernetes will update the default static pod with the yaml file automatically.

### 2.1 Add Policy config file parameter in scheduler arguments

```yaml
- --policy-config-file=/etc/kubernetes/scheduler-policy-config.json
```

### 2.2 Add volume mount into Pod Spec

```yaml
- mountPath: /etc/kubernetes/scheduler-policy-config.json
  name: scheduler-policy-config
  readOnly: true
```

```yaml
- hostPath:
      path: /etc/kubernetes/scheduler-policy-config.json
      type: FileOrCreate
  name: scheduler-policy-config
```

## 3\. Deploy Device Plugin

```bash
wget https://raw.githubusercontent.com/AliyunContainerService/gpushare-device-plugin/master/device-plugin-rbac.yaml
kubectl create -f device-plugin-rbac.yaml
wget https://raw.githubusercontent.com/AliyunContainerService/gpushare-device-plugin/master/device-plugin-ds.yaml
kubectl create -f device-plugin-ds.yaml
```

> Notice: please remove default GPU device plugin, for example, if you are using [nvidia-device-plugin](https://github.com/NVIDIA/k8s-device-plugin/blob/v1.11/nvidia-device-plugin.yml), you can run `kubectl delete ds -n kube-system nvidia-device-plugin-daemonset` to delete.

## 4\. Add gpushare node labels to the nodes requiring GPU sharing
You need to add a label "gpushare=true" to all node where you want to install device plugin because the device plugin is deamonset. 
```bash
kubectl label node <target_node> gpushare=true
```

For example:

```bash
kubectl label node mynode gpushare=true
```

## 5\. Install Kubectl extension


### 5.1 Install kubectl 1.12 or above
You can download and install `kubectl` for linux

```bash
curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.12.1/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/bin/kubectl
```

### 5.2 Download and install the kubectl extension

```bash
cd /usr/bin/
wget https://github.com/AliyunContainerService/gpushare-device-plugin/releases/download/v0.3.0/kubectl-inspect-gpushare
chmod u+x /usr/bin/kubectl-inspect-gpushare
```
