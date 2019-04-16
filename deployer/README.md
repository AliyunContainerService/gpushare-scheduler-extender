## Deployment

Just run:

```
git clone https://github.com/AliyunContainerService/gpushare-scheduler-extender.git
cd gpushare-scheduler-extender/deployer/chart
helm install --set kubeVersion=1.11.5 gpushare-installer
```