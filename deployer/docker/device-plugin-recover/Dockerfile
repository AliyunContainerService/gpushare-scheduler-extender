FROM debian:bullseye-slim
RUN echo \
    deb [arch=amd64]  http://mirrors.aliyun.com/debian/ bullseye main non-free contrib\
    > /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y curl tzdata iproute2 bash &&  \
 	rm -rf /var/cache/apt/* && \
 	cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
 	echo "Asia/Shanghai" >  /etc/timezone && \
 	mkdir -p /dp-evict
ADD dp-evict /dp-evict
RUN chmod -R +x /dp-evict
