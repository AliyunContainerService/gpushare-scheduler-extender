FROM python:2.7-alpine

MAINTAINER 2270020588@qq.com

ADD ./run.py /root/run.py

RUN mkdir /usr/local/k8s-schd-extender && \
	apk add  --update tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" >  /etc/timezone

RUN  pip install  pyyaml

CMD ["python","/root/run.py"]
