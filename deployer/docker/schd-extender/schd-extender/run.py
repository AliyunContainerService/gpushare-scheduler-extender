#!/usr/bin/env python
#-*- coding: utf-8 -*-
import time
import sys
import re
import os
import yaml
import io
import json
import copy
import shutil
import hashlib

scheduler_policy_volume = {
	"hostPath": {
		"path": "/etc/kubernetes/scheduler-policy-config.json"
	},
	"name": "scheduler-policy-config"
}

scheduler_policy = {
	"apiVersion": "v1",
	"extenders": [],
	 "kind": "Policy"	
}

class ConfigTool():
	def __init__(self):
		self.config = read_config()
		self.policy_md5 = ""
		self.manifest_md5 = ""

	def run(self):
		while True:
			time.sleep(self.config["time_interval"])
			self.update()

	# update scheduler-policy-config.json when its' md5 is changed
	def update_policy(self):
		policy_file = scheduler_policy_volume["hostPath"]["path"]
		if not os.path.exists(policy_file):
			self.change_ip()
			return True
		if os.path.isdir(policy_file):
			Print("error","there is a directory whose name is",policy_file)	
			shutil.rmtree(policy_file)
			self.change_ip()
			return True
		policy_md5 = get_md5(policy_file)
		if self.policy_md5 == policy_md5:
			return False
		self.change_ip()
		return True

	# update /etc/kubernetes/manifests/kube-scheduler.yaml 
	def update(self):
		policy_file = scheduler_policy_volume["hostPath"]["path"]
		should_update  = self.update_policy()
		self.policy_md5 = get_md5(policy_file)
		kube_file = self.config["scheduler_yaml_file"]
		config = read_file(kube_file) 
		if not config:
			Print("error","failed to read content",kube_file,",skip to update")
			return 
		manifest_md5 = get_md5(kube_file)
		if self.manifest_md5 != manifest_md5:
			Print("debug","file",kube_file,"has been changed")
			should_update = True
		if not should_update:
			return 
		Print("debug","start to update",kube_file)
		config = self.update_volumes(config)
		config = self.update_volume_mounts(config)
		config = self.update_command(config)
		config = self.update_annotation(config)
		with io.open(kube_file,'w',encoding="UTF-8") as update:
			yaml.dump(config,update)
		self.manifest_md5 = get_md5(kube_file)
	

	def update_volumes(self,config):
		if "volumes" not in config["spec"]:
			Print("debug","field 'volumes' not found,create it")
			config["spec"]["volumes"] = list()
		volumes = list()
		for volume in config["spec"]["volumes"]:
			# if volume name is 'scheduler-policy-config',skip it
			if volume["name"] == "scheduler-policy-config":
				continue
			volumes.append(volume)
		volumes.append(scheduler_policy_volume)
		config["spec"]["volumes"] = volumes
		return config

	def change_ip(self):
		Print("debug","file",scheduler_policy_volume["hostPath"]["path"],"has been changed,start to update")
		policy_config = None
		policy_file = scheduler_policy_volume["hostPath"]["path"]
		if os.path.exists(policy_file):
			policy_config = read_file(policy_file)
		else:
			policy_config = self.config["policy_config"]
		if not policy_config:
			Print("error","failed to read content of",policy_file)
			return 					
		if "extenders" not in policy_config:
			policy_config["extenders"] = list()
		extender_configs = list()
		for policy in policy_config["extenders"]:
			if 'managedResources' not in policy:
				extender_configs.append(policy)
				continue
			skip = False
			for resource in policy['managedResources']:
				if resource['name']  == self.config["policy_config"]["extenders"][0]["managedResources"][0]["name"]:
					skip = True
					break
			if not skip:
				extender_configs.append(policy)
		extender_configs.append(self.config["policy_config"]["extenders"][0])
		policy_config["extenders"] = extender_configs
		for policy in policy_config["extenders"]:
			policy["urlPrefix"] = change_ip(policy["urlPrefix"],self.config["node_name"]) 
		jsonstr = json.dumps(policy_config,sort_keys=True, indent=4, separators=(',', ': '))
		with open(policy_file,'w') as f:
			f.write(jsonstr)
			
	def update_volume_mounts(self,config):
		for container in config["spec"]["containers"]:
			if "command" not in container:
				continue
			if len(container["command"]) == 0:
				continue
			if container["command"][0].strip(" ") != "kube-scheduler":
				continue
			if "volumeMounts" not in container:
				container["volumeMounts"] = list()
			mounts = list()
			for mount in container["volumeMounts"]:
				if mount["name"] != scheduler_policy_volume["name"]:
					mounts.append(mount)
			mounts.append({"name": scheduler_policy_volume["name"],"mountPath": scheduler_policy_volume["hostPath"]["path"]})
			container["volumeMounts"] = mounts
		return config
		
	def update_command(self,config):
		for container in config["spec"]["containers"]:
			if "command" not in container:
				continue
			if len(container["command"]) == 0:
				continue
			if container["command"][0].strip(" ") != "kube-scheduler":
				continue
			should_add = True
			options = list()
			for option in container["command"]:
				if option.find("--policy-config-file=") != -1:
					continue
				options.append(option)
			options.append("--policy-config-file=" + scheduler_policy_volume["hostPath"]["path"])
			container["command"] = options
		return config

	def update_annotation(self,config):
		if "annotations" not in config["metadata"]:
			config["metadata"]["annotations"] = dict()
		annotations = dict()
		for key,value in config["metadata"]["annotations"].items():
			if key != "deployment.kubernetes.io/revision":
				annotations[key] = value
		timestr = time.strftime('%Y-%m-%d_%H:%M:%S',time.localtime())
		annotations["deployment.kubernetes.io/revision"] = timestr 
		config["metadata"]["annotations"] = annotations
		return config
			
			
			
def read_file(config_file):
	try:
		name,ext = os.path.splitext(config_file)
		with open(config_file,"r") as config:
			if ext == ".yaml" or ext == ".yml":
				return yaml.load(config,Loader=yaml.FullLoader)
			elif ext == ".json":
				return json.load(config)
			else:
				Print("error","unknown file type of",config_file,",it should be yaml or json")
				return None
	except Exception as err:
		Print("error","failed to read",config_file,",reason:",str(err))
		return None

def set_value(env,value):
	if not os.getenv(env) or os.getenv(env) == "":
		return value
	return  os.getenv(env)

def read_config():
	config = dict()
	policy_config = dict()
	kube_yaml = set_value('SCHEDULER_STATIC_POD_YAML','/etc/kubernetes/manifests/kube-scheduler.yaml')
	node_name = set_value('NODE_IP','UNKNOWN')
	time_interval = set_value('TIME_INTERVAL_FOR_CHECKING','3')
	prioritize_verb = set_value('PRIORITIZE_VERB','sort')
	bind_verb = set_value('BIND_VERB','bind')
	enable_https = set_value('ENABLE_HTTPS','false')
	filter_verb = set_value('FILTER_VERB','filter')
	ignorable = set_value("IGNORABLE",'false')
	node_cache_capable = set_value('NODE_CACHE_CAPABLE',"true")
	url_prefix = set_value('URL_PREFIX',"http://127.0.0.1:32766/gpushare-scheduler")
	weight = set_value('WEIGHT','-9999')
	resource_name = set_value('RESOURCE_NAME','aliyun.com/gpu-mem')
	ignored_by_scheduler = set_value('IGNORED_BY_SCHEDULER','false')
	policy_config['urlPrefix'] = url_prefix
	policy_config['filterVerb'] = filter_verb
	policy_config['bindVerb'] = bind_verb
	if enable_https == 'false':
		policy_config['enableHttps'] = False
	else:
		policy_config['enableHttps'] = True
	if ignorable == 'false':
		policy_config['ignorable'] = False
	else:
		policy_config['ignorable'] = True
	resource = dict()
	resource['name'] = resource_name
	if ignored_by_scheduler == 'false':
		resource['ignoredByScheduler'] = False
	else:
		resource['ignoredByScheduler'] = True
	policy_config['managedResources'] = list()
	policy_config['managedResources'].append(resource)
	if node_cache_capable == 'false':
		policy_config['nodeCacheCapable'] = False
	else:
		policy_config['nodeCacheCapable'] = True
	if weight != "-9999":
		policy_config["weight"] = int(weight)
	config["node_name"] = node_name
	config['time_interval'] = int(time_interval)
	config["scheduler_yaml_file"] =  kube_yaml
	config["policy_config"] = scheduler_policy	 
	config["policy_config"]["extenders"].append(policy_config)
	return config

def get_md5(file_path):
    try:
        md5_obj = hashlib.md5()
        with open(file_path,'r',encoding='utf-8') as myfile:
            md5_obj.update(myfile.read().encode("utf-8"))
        hash_code = md5_obj.hexdigest()
        md5 = str(hash_code).lower()
        return md5
    except Exception as err:
        Print("error","get hash code failed for file,reason:",str(err))
        return None

def Print(level,*messages):
	timestr = time.strftime('%Y-%m-%d %H:%M:%S',time.localtime())
	msg = ' '.join(messages)
	print(timestr,'  ',level.upper(),'  ',msg)				

def change_ip(url,ip):
	pattern = re.compile(r'[http|https]://(.*?):.*')
	old = pattern.findall(url)
	if not old:
		return url
	if len(old) == 0:
		return url
	return url.replace(old[0],ip,1)

def start():
	ct = ConfigTool()
	ct.run()

start()
