import fire
import yaml


def modify_manifest(requestcpu, requestmemory, limitcpu, limitmemory, image):
    with open('nvidia-device-plugin.yml', 'r+') as f:
        data = yaml.load(f, yaml.FullLoader)

        if requestcpu != '':
            data['spec']['containers'][0]['resources']['requests']['cpu'] = requestcpu

        if requestmemory != '':
            data['spec']['containers'][0]['resources']['requests']['memory'] = requestmemory

        if limitcpu != '':
            data['spec']['containers'][0]['resources']['limits']['cpu'] = limitcpu

        if limitmemory != '':
            data['spec']['containers'][0]['resources']['limits']['memory'] = limitmemory

        if image != '':
            data['spec']['containers'][0]['image'] = image

        yaml.dump(data, f)


fire.Fire(modify_manifest)
