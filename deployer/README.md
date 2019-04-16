## Install GPU Sharing in Alibaba Cloud Kubernetes Service

1.Just run:

```
git clone https://github.com/AliyunContainerService/gpushare-scheduler-extender.git
cd gpushare-scheduler-extender/deployer/chart
helm install --set kubeVersion=1.11.5 gpushare-installer
```

2.Add gpushare node labels to the nodes requiring GPU sharing

```bash
kubectl label node <target_node> gpushare=true
```

For example:

```bash
kubectl label no mynode gpushare=true
```

3.Install Kubectl extension

4.Install kubectl 1.12 or above
You can download and install `kubectl` for linux

```bash
curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.12.1/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/bin/kubectl
```

5.Download and install the kubectl extension

```bash
cd /usr/bin/
wget https://github.com/AliyunContainerService/gpushare-device-plugin/releases/download/v0.2.0/kubectl-inspect-gpushare
chmod u+x /usr/bin/kubectl-inspect-gpushare
```