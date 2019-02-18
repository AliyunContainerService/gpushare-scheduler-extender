# User Guide

> Notice: Kubernetes provides GPU sharing scheduling capability, which is only a scheduling mechanism that
guarantees that devices can not be “oversubscribed” (at the scheduling level)， but cannot in any
measure enforce that at the runtime level. For now, you have to take care of isolation by yourself now. 

1. Query the allocation status of the shared GPU

```
# kubectl inspect gpushare
NAME           IPADDRESS     GPU0(Allocated MiB/Total MiB)  GPU Memory
i-2ze0gl97vw1  192.168.0.72  7606/7606                      7606/7606
i-2ze0gl97vw2  192.168.0.73  3803/7606                      3803/7606
i-2ze0gl97vw3  192.168.0.71  7606/7606                      7606/7606
---------------------------------------------------------------------------------------
Allocated/Total GPU Memory In Cluster:
19015/22818 (83%)
```

> For more details, please run `kubectl inspect gpushare -d`

2. To request GPU sharing, you just need to specify `aliyun.com/gpu-mem`

```
apiVersion: apps/v1
kind: Deployment

metadata:
  name: binpack-1
  labels:
    app: binpack-1

spec:
  replicas: 1

  selector: # define how the deployment finds the pods it mangages
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
            # MiB
            aliyun.com/gpu-mem: 3803
```

> Notice that the GPU memory of each GPU is 7606 MiB, 3803 MiB indicates half of the GPU.

3\. From the following environment variables,the application can limit the GPU usage by using CUDA API or framework API, such as Tensorflow

```
# The total amount of GPU memory on the current device (MiB)
ALIYUN_COM_GPU_MEM_DEV=7606 

# The GPU Memory of the container(MiB)
ALIYUN_COM_GPU_MEM_CONTAINER=3803
```

Limit GPU memory by setting fraction through TensorFlow API

```
fraction = round( 3803 * 0.7 / 7606 , 1 )
config = tf.ConfigProto()
config.gpu_options.per_process_gpu_memory_fraction = fraction
sess = tf.Session(config=config)
# Runs the op.
while True:
	sess.run(c)
```

> 0.7 is because tensorflow control gpu memory is not accurate, it is recommended to multiply by 0.7 to ensure that the upper limit is not exceeded.