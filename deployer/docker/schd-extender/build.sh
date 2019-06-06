#!/usr/bin/env bash
set -xe

BASEDIR=$(dirname "$0")
echo "build docker images in $BASEDIR"

TIMESTAMP=$(date +%Y%m%d%H%M)

cd $BASEDIR

# docker build --no-cache -t $IMAGE -f $FILE $BASEDIR
docker build -t registry.cn-hangzhou.aliyuncs.com/acs/schd-extender-deployer:$TIMESTAMP .

docker tag registry.cn-hangzhou.aliyuncs.com/acs/schd-extender-deployer:$TIMESTAMP cheyang/schd-extender-deployer:$TIMESTAMP

docker push registry.cn-hangzhou.aliyuncs.com/acs/schd-extender-deployer:$TIMESTAMP

