FROM registry.aliyuncs.com/acs/alpine:3.3
RUN apk add --update curl tzdata iproute2 bash python3 py-pip &&  \
 	rm -rf /var/cache/apk/* && \
 	cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
 	echo "Asia/Shanghai" >  /etc/timezone && \
 	mkdir -p /dp-evict && \
 	pip install PyYaml click fire

ADD dp-modify /dp-modify
RUN chmod -R +x /dp-modify
