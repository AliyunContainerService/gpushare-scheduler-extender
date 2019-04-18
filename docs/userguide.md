# User Guide

> Notice: Kubernetes provides GPU sharing scheduling capability, which is only a scheduling mechanism that
guarantees that devices can not be “oversubscribed” (at the scheduling level), but cannot in any
measure enforce that at the runtime level. For now, you have to take care of isolation by yourself. 

1. Query the allocation status of the shared GPU

```bash
# kubectl inspect gpushare
NAME                                IPADDRESS     GPU0(Allocated/Total)  GPU Memory(GiB)
cn-shanghai.i-uf61h64dz1tmlob9hmtb  192.168.0.71  6/15                   6/15
cn-shanghai.i-uf61h64dz1tmlob9hmtc  192.168.0.70  3/15                   3/15
------------------------------------------------------------------------------
Allocated/Total GPU Memory In Cluster:
9/30 (30%)
```

> For more details, please run `kubectl inspect gpushare -d`

2. To request GPU sharing, you just need to specify `aliyun.com/gpu-mem`

```yaml
apiVersion: apps/v1beta1
kind: StatefulSet

metadata:
  name: binpack-1
  labels:
    app: binpack-1

spec:
  replicas: 3
  serviceName: "binpack-1"
  podManagementPolicy: "Parallel"
  selector: # define how the deployment finds the pods it manages
    matchLabels:
      app: binpack-1

  template: # define the pods specifications
    metadata:
      labels:
        app: binpack-1

    spec:
      containers:
      - name: binpack-1
        image: cheyang/gpu-player:v2
        resources:
          limits:
            # GiB
            aliyun.com/gpu-mem: 3
```

> Notice that the GPU memory of each GPU is 3 GiB, 3 GiB indicates one third of the GPU.

3\. From the following environment variables,the application can limit the GPU usage by using CUDA API or framework API, such as Tensorflow

```bash
# The total amount of GPU memory on the current device (GiB)
ALIYUN_COM_GPU_MEM_DEV=15 

# The GPU Memory of the container (GiB)
ALIYUN_COM_GPU_MEM_CONTAINER=3
```

Limit GPU memory by setting fraction through TensorFlow API

```python
fraction = round( 3 * 0.7 / 15 , 1 )
config = tf.ConfigProto()
config.gpu_options.per_process_gpu_memory_fraction = fraction
sess = tf.Session(config=config)
# Runs the op.
while True:
	sess.run(c)
```

> 0.7 is because tensorflow control gpu memory is not accurate, it is recommended to multiply by 0.7 to ensure that the upper limit is not exceeded.