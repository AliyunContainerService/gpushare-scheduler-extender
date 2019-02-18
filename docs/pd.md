## Problem Determination

1. If there is no way to find the gpushare node through `kubectl inspect gpushare`：

1.1 kubectl get po -n kube-system -o=wide | grep gpushare-device

1.2 kubecl logs -n kube-system <pod_name>