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

class ConfigTool():
	def __init__(self):
		self.config = read_config()
		self.write_by_me = False
		self.policy_md5 = None
		self.define_md5 = None

	def listen_directory(self):
		file_path = os.path.join(self.config["kube_dir"])
		return file_path
	def run(self):
		try:
			while True:
				time.sleep(self.config["time_interval"])
				self.file_mtime()
		except Exception,err:
			Print("error","some error occurred,detail:",str(err))
			sys.exit(3)
	def file_mtime(self):
		try:
			policy_file = os.path.join(self.config["config_dir"],self.config["policy_file"])
			dst_policy_file = os.path.join(self.config["kube_dir"],self.config["policy_file"])
			define_yaml = os.path.join(self.config["kube_dir"],"manifests",self.config["define_yaml"])
			should_update = False
			force_update = False
			if not os.path.exists(dst_policy_file):
				Print("error",dst_policy_file,"not found,create it")
				self.update_scheduler_yaml(True)
				self.define_md5 = get_md5(define_yaml)
				self.policy_md5 = get_md5(policy_file)
				return 
			if not os.path.exists(policy_file):
				Print("error",policy_file,"not found")
				os.exit(2)
				return
			if not os.path.exists(define_yaml):
				Print("error",define_yaml," not found")
				return
			policy_new_md5 = get_md5(policy_file)
			define_new_md5 = get_md5(define_yaml)
			if policy_new_md5 != self.policy_md5:
				Print("debug",policy_file,"has been modified,we will update",define_yaml)
				should_update = True	
				force_update = True
			if  define_new_md5 != self.define_md5:
				Print("debug",define_yaml,"has been modified,we will update it")
				should_update = True
			if not should_update:
				return
			Print("debug","********************************* update start *********************")
			self.update_scheduler_yaml(force_update)
			self.define_md5 = get_md5(define_yaml)
			self.policy_md5 = get_md5(policy_file)
		except Exception,err:
			Print("error","get error:",str(err))
			return
	def change_ip_of_config(self,file_path,node_ip):
		try:
			with open(file_path,'r') as jfile:
				config = json.load(jfile)
			if not config["extenders"][0].has_key("urlPrefix"):
				Print("error","not found field: .extenders[0].urlPrefix,please check the content of",file_path)
				return None
			url = config["extenders"][0]["urlPrefix"]
			new_url = re.sub(r'://.*:','://'+ node_ip + ":",url)
			config["extenders"][0]["urlPrefix"] = new_url
			return config
		except Exception,err:
			Print("error","change ip in file",file_path,"failed,reason:",str(err))
			return None

	
	def copy_and_backup(self):
		try:
			timetr = time.strftime('%Y-%m-%d_%H:%M:%S',time.localtime())
			src_policy_file = os.path.join(self.config["config_dir"],self.config["policy_file"])
			dst_policy_file = os.path.join(self.config["kube_dir"],self.config["policy_file"])
			policy_file = '.'.join(self.config["policy_file"].split(".")[0:-1]) + "." + timetr + "." + self.config["policy_file"].split(".")[-1] 			
			back_policy_file = os.path.join(self.config["kube_dir"],"manifests_backup",policy_file)
			src_define_yaml = os.path.join(self.config["kube_dir"],"manifests",self.config["define_yaml"])
			define_yaml = '.'.join(self.config["define_yaml"].split(".")[0:-1]) + "." + timetr + "." + self.config["define_yaml"].split(".")[-1] 
			back_define_yaml = os.path.join(self.config["kube_dir"],"manifests_backup",define_yaml)
			if os.path.exists(dst_policy_file):
				shutil.copy2(dst_policy_file,back_policy_file)
			if os.path.exists(src_define_yaml):
				shutil.copy2(src_define_yaml,back_define_yaml)
			policy_config = self.change_ip_of_config(src_policy_file,self.config["node_ip"])
			if not policy_config:
				Print("error","copy policy file failed")
				return False
			jsonstr = json.dumps(policy_config,sort_keys=True, indent=4, separators=(',', ': '))
			with open(dst_policy_file,'w') as jfile:
				jfile.write(jsonstr)
			self.delete_old_backups(
				os.path.join(self.config["kube_dir"],"manifests_backup"),
				'.'.join(self.config["define_yaml"].split(".")[0:-1]),
				'.'.join(self.config["policy_file"].split(".")[0:-1])	
			)
			return True
		except Exception,err:
			Print("error","copy and backup failed,reason:",str(err))
			return False
		else:
			Print("debug","copy and backup succeed")
	def delete_old_backups(self,file_path,*target):
		yaml_files = list()
		json_files = list()
		dir_list = os.listdir(file_path)
		for fi in dir_list:
			if fi.find(target[0]) == 0:
				yaml_files.append(fi)
			if fi.find(target[1]) == 0:
				json_files.append(fi)
		yaml_files = sorted(yaml_files,key=lambda x: os.path.getctime(os.path.join(file_path,x)))
		json_files = sorted(json_files,key=lambda x: os.path.getctime(os.path.join(file_path,x)))
		yaml_files.reverse()
		json_files.reverse()
		if len(yaml_files) > 3:
			for fi in yaml_files[3:]:
				os.remove(os.path.join(file_path,fi))
				Print("debug","remove old backup file",os.path.join(file_path,fi))
		if len(json_files) > 3:
			for fi in json_files[3:]:
				os.remove(os.path.join(file_path,fi))
				Print("debug","remove old backup file",os.path.join(file_path,fi))
	def update_volumes(self,defines):
		try:
			# checking need update volume or not
			if not defines["spec"].has_key("volumes"):
				defines["spec"]["volumes"] = list()
			volumes = defines["spec"]["volumes"]
			add_volume = True
			volume_name = self.config["volume_name"]
			found_name = False
			volume_ind = 0
			old_volumes = list() 
			for ind in range(len(volumes)):
				if volumes[ind]["name"] == volume_name:
					old_volumes.append(ind)
			cur = 0
			for ind in old_volumes:
				del volumes[ind-cur]
				cur = cur + 1
			policy_volume = dict()
			policy_volume["name"] = volume_name
			policy_volume["hostPath"] = dict()
			policy_volume["hostPath"]["type"] = "FileOrCreate"
			policy_volume["hostPath"]["path"] = os.path.join(self.config["kube_dir"],self.config["policy_file"])
			volumes.append(copy.deepcopy(policy_volume))
			Print("debug","update volumes succeed")
			return defines
		except Exception,err:
			Print("error","failed to update volumes,reason:",str(err))
			return None
	def update_volume_mounts(self,defines):
		try:
			# update volume mounts
			if not defines["spec"]["containers"][0].has_key("volumeMounts"):
				defines["spec"]["containers"][0]["volumeMounts"] = list()
			volume_mounts = defines["spec"]["containers"][0]["volumeMounts"]
			volume_name = self.config["volume_name"]
			old_mounts= list()
			for ind in range(len(volume_mounts)):
				if volume_mounts[ind]["name"] == volume_name:
					old_mounts.append(ind)
			cur = 0
			for ind in old_mounts:
				del volume_mounts[ind-cur]
				cur = cur + 1
			mount_info = dict()
			mount_info["mountPath"] = os.path.join(self.config["kube_dir"],self.config["policy_file"])
			mount_info["name"] = volume_name
			mount_info["readOnly"] = True
			volume_mounts.append(copy.deepcopy(mount_info))
			Print("debug","update volume mounts succeed")
			return defines
		except Exception,err:
			Print("error","failed to update volume mounts,reason:",str(err))
			return None
	def update_annotation(self,defines):
		file_path = os.path.join(self.config["kube_dir"],"manifests",self.config["define_yaml"])
		timetr = time.strftime('%Y-%m-%d_%H:%M:%S',time.localtime())
		# check annotation has tag 'deployment.kubernetes.io/revision' or not
		if not defines["metadata"].has_key("annotations"):
			defines["metadata"]["annotations"] = dict()
		annotations = defines["metadata"]["annotations"]
		# update annotation
		annotations["deployment.kubernetes.io/revision"] = timetr
		Print("debug","add annotation deployment.kubernetes.io/revision=" + timetr,"to",file_path)
		return defines
	def update_command(self,defines):
		# check command has option --policy-config-file= or not
		file_path = os.path.join(self.config["kube_dir"],"manifests",self.config["define_yaml"])
		if not defines["spec"]["containers"][0].has_key("command"):
			defines["spec"]["containers"][0]["command"]=list()
		commands = defines["spec"]["containers"][0]["command"]
		del_ind = -1
		old_opts = list()
		for ind in range(len(commands)):
			if commands[ind].strip(" ").find("--policy-config-file=") == 0:
				old_opts.append(ind)
		cur = 0
		for ind in old_opts:
			del commands[ind - cur]
			cur = cur + 1
		commands.append("--policy-config-file=" + os.path.join(self.config["kube_dir"],self.config["policy_file"])) 
		Print("debug","add command option --policy-config-file="+ os.path.join(self.config["kube_dir"],self.config["policy_file"]),"to",file_path)
		return defines
	def update_scheduler_yaml(self,force_update):
		try:
			# config_changed is used to check the scheduler yaml file is changed or not 
			back_dir = os.path.join(self.config["kube_dir"],"manifests_backup")
			if not os.path.exists(back_dir):
				os.makedirs(back_dir)
			scheduler_define_yaml = self.config["define_yaml"]
			file_path = os.path.join(self.config["kube_dir"],"manifests",scheduler_define_yaml)
			current_md5 = get_md5(file_path)
			if current_md5 == self.define_md5 and not force_update: 
				Print("debug","content of",file_path,"not changed,skip to handle it")
				return 
			self.define_md5 = current_md5
			defines = read_scheduler_yaml(file_path)
			old = copy.deepcopy(defines)
			defines = self.update_annotation(defines)
			if not defines:
				Print("error","update",file_path,"failed")
				return 
			defines = self.update_command(defines)
			if not defines:
				Print("error","update",file_path,"failed")
				return 
			defines = self.update_volume_mounts(defines)	
			if not defines:
				Print("error","update",file_path,"failed")
				return 
			defines = self.update_volumes(defines)
			if not defines:
				Print("error","update",file_path,"failed")
				return 
			if not self.copy_and_backup():
				return
			with io.open(file_path,'w',encoding="UTF-8") as update:
				yaml.dump(defines,update)
				self.write_by_me = True
		except Exception,err:
			Print("error","update","failed,reason:",str(err)) 
			sys.exit(1)

def read_scheduler_yaml(yaml_file):
	try:
		name = os.path.basename(yaml_file)
		with open(yaml_file,'r') as myfile:
			config_obj = yaml.load(myfile,Loader=yaml.FullLoader)
		return config_obj
	except Exception,err:
		Print("error","read",yaml_file,"failed,reason:",str(err))
		sys.exit(1)

def read_config():
	config = dict()
	config["kube_dir"] = "/etc/kubernetes"
	config["config_dir"] = "/usr/local/k8s-schd-extender"
	policy_file = os.getenv('SCHEDULER_POLICY_FILE_NAME')
	if policy_file == None or policy_file == "":
		Print("debug","env SCHEDULER_POLICY_FILE_NAME not set,we use the default name scheduler-policy-config.json")
		policy_file = "scheduler-policy-config.json"
	config["policy_file"] = policy_file
	scheduler_define_yaml = os.getenv("SCHEDULER_DEFINE_YAML")
	if scheduler_define_yaml == None or scheduler_define_yaml == "":
		scheduler_define_yaml = "kube-scheduler.yaml"
		Print("debug","env SCHEDULER_DEFINE_YAML not set,we use the default name kube-scheduler.yaml")
	volume_name = os.getenv("SCHEDULER_VOLUME_NAME")
	if volume_name == None or volume_name == "":
		volume_name = "scheduler-policy-with-extender"
		Print("debug","SCHEDULER_VOLUME_NAME not set,we use the default name scheduler-policy-config")
	node_ip = os.getenv("NODE_IP")
	if node_ip == None or node_ip == "":
		node_ip = "127.0.0.1"
	config["node_ip"] = node_ip
	time_interval = os.getenv("CHECK_FILE_INTERVAL")
	if time_interval == None or time_interval == "":
		time_interval = "3"
	config["time_interval"] = int(time_interval)
	config["volume_name"] = volume_name
	config["define_yaml"] = scheduler_define_yaml
	return config

def get_md5(file_path):
    try:
        md5_obj = hashlib.md5()
        with open(file_path,'r') as myfile:
            md5_obj.update(myfile.read())
        hash_code = md5_obj.hexdigest()
        md5 = str(hash_code).lower()
        return md5
    except Exception,err:
        Print("error","get hash code failed for file,reason:",str(err))
        return None

def Print(level,*messages):
	timestr = time.strftime('%Y-%m-%d %H:%M:%S',time.localtime())
	msg = ' '.join(messages)
	print timestr,'  ',level.upper(),'  ',msg				

def start():
	ct = ConfigTool()
	ct.run()

start()
