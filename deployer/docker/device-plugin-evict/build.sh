#!/usr/bin/env bash
set -xe

BASEDIR=$(dirname "$0")
echo "build docker images in $BASEDIR"

TIMESTAMP=$(date +%Y%m%d%H%M)

cd $BASEDIR

# docker build --no-cache -t $IMAGE -f $FILE $BASEDIR
docker build -t registry.cn-hangzhou.aliyuncs.com/acs/device-plugin-evict:$TIMESTAMP .

docker tag registry.cn-hangzhou.aliyuncs.com/acs/device-plugin-evict:$TIMESTAMP cheyang/device-plugin-evict:$TIMESTAMP
