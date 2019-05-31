#!/usr/bin/env bash
set -xe

BASEDIR=$(dirname "$0")
echo "build docker images in $BASEDIR"

TIMESTAMP=$(date +%Y%m%d%H%M)

cd $BASEDIR

# docker build --no-cache -t $IMAGE -f $FILE $BASEDIR
docker build --network=host -t registry.cn-hangzhou.aliyuncs.com/acs/nvidia-device-plugin-evict:$TIMESTAMP .

docker push registry.cn-hangzhou.aliyuncs.com/acs/nvidia-device-plugin-evict:$TIMESTAMP

echo registry.cn-hangzhou.aliyuncs.com/acs/nvidia-device-plugin-evict:$TIMESTAMP
