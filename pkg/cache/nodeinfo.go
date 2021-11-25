package cache

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"

	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	OptimisticLockErrorMsg = "the object has been modified; please apply your changes to the latest version and try again"
)

// NodeInfo is node level aggregated information.
type NodeInfo struct {
	name           string
	node           *v1.Node
	devs           map[int]*DeviceInfo
	gpuCount       int
	gpuTotalMemory int
	rwmu           *sync.RWMutex
}

type DeviceIndexAvailableMem struct {
	Idx  int
	AvailableMem uint
}


// Create Node Level
func NewNodeInfo(node *v1.Node) *NodeInfo {
	log.Printf("debug: NewNodeInfo() creates nodeInfo for %s", node.Name)

	devMap := map[int]*DeviceInfo{}
	for i := 0; i < utils.GetGPUCountInNode(node); i++ {
		devMap[i] = newDeviceInfo(i, uint(utils.GetTotalGPUMemory(node)/utils.GetGPUCountInNode(node)))
	}

	if len(devMap) == 0 {
		log.Printf("warn: node %s with nodeinfo %v has no devices",
			node.Name,
			node)
	}

	return &NodeInfo{
		name:           node.Name,
		node:           node,
		devs:           devMap,
		gpuCount:       utils.GetGPUCountInNode(node),
		gpuTotalMemory: utils.GetTotalGPUMemory(node),
		rwmu:           new(sync.RWMutex),
	}
}

// Only update the devices when the length of devs is 0
func (n *NodeInfo) Reset(node *v1.Node) {
	n.gpuCount = utils.GetGPUCountInNode(node)
	n.gpuTotalMemory = utils.GetTotalGPUMemory(node)
	n.node = node
	if n.gpuCount == 0 {
		log.Printf("warn: Reset for node %s but the gpu count is 0", node.Name)
	}

	if n.gpuTotalMemory == 0 {
		log.Printf("warn: Reset for node %s but the gpu total memory is 0", node.Name)
	}

	if len(n.devs) == 0 && n.gpuCount > 0 {
		devMap := map[int]*DeviceInfo{}
		for i := 0; i < utils.GetGPUCountInNode(node); i++ {
			devMap[i] = newDeviceInfo(i, uint(n.gpuTotalMemory/n.gpuCount))
		}
		n.devs = devMap
	}
	log.Printf("info: Reset() update nodeInfo for %s with devs %v", node.Name, n.devs)
}

func (n *NodeInfo) GetName() string {
	return n.name
}

func (n *NodeInfo) GetDevs() []*DeviceInfo {
	devs := make([]*DeviceInfo, n.gpuCount)
	for i, dev := range n.devs {
		devs[i] = dev
	}
	return devs
}

func (n *NodeInfo) GetNode() *v1.Node {
	return n.node
}

func (n *NodeInfo) GetTotalGPUMemory() int {
	return n.gpuTotalMemory
}

func (n *NodeInfo) GetGPUCount() int {
	return n.gpuCount
}

func (n *NodeInfo) removePod(pod *v1.Pod) {
	n.rwmu.Lock()
	defer n.rwmu.Unlock()

	id := utils.GetGPUIDFromAnnotation(pod)
	if id >= 0 {
		dev, found := n.devs[id]
		if !found {
			log.Printf("warn: Pod %s in ns %s failed to find the GPU ID %d in node %s", pod.Name, pod.Namespace, id, n.name)
		} else {
			dev.removePod(pod)
		}
	} else {
		log.Printf("warn: Pod %s in ns %s is not set the GPU ID %d in node %s", pod.Name, pod.Namespace, id, n.name)
	}
}

// Add the Pod which has the GPU id to the node
func (n *NodeInfo) addOrUpdatePod(pod *v1.Pod) (added bool) {
	n.rwmu.Lock()
	defer n.rwmu.Unlock()

	id := utils.GetGPUIDFromAnnotation(pod)
	log.Printf("debug: addOrUpdatePod() Pod %s in ns %s with the GPU ID %d should be added to device map",
		pod.Name,
		pod.Namespace,
		id)
	if id >= 0 {
		dev, found := n.devs[id]
		if !found {
			log.Printf("warn: Pod %s in ns %s failed to find the GPU ID %d in node %s", pod.Name, pod.Namespace, id, n.name)
		} else {
			dev.addPod(pod)
			added = true
		}
	} else {
		log.Printf("warn: Pod %s in ns %s is not set the GPU ID %d in node %s", pod.Name, pod.Namespace, id, n.name)
	}
	return added
}

// check if the pod can be allocated on the node
func (n *NodeInfo) Assume(pod *v1.Pod) (allocatable bool) {
	allocatable = false

	n.rwmu.RLock()
	defer n.rwmu.RUnlock()

	availableGPUs := n.getAvailableGPUs()
	reqGPU := uint(utils.GetGPUMemoryFromPodResource(pod))
	log.Printf("debug: AvailableGPUs: %v in node %s", availableGPUs, n.name)

	if len(availableGPUs) > 0 {
		for devID := 0; devID < len(n.devs); devID++ {
			availableGPU, ok := availableGPUs[devID]
			if ok {
				if availableGPU >= reqGPU {
					allocatable = true
					break
				}
			}
		}
	}

	return allocatable

}

func (n *NodeInfo) Allocate(clientset *kubernetes.Clientset, pod *v1.Pod) (err error) {
	var newPod *v1.Pod
	n.rwmu.Lock()
	defer n.rwmu.Unlock()
	log.Printf("info: Allocate() ----Begin to allocate GPU for gpu mem for pod %s in ns %s----", pod.Name, pod.Namespace)
	// 1. Update the pod spec
	devId, found := n.allocateGPUID(pod)
	if found {
		log.Printf("info: Allocate() 1. Allocate GPU ID %d to pod %s in ns %s.----", devId, pod.Name, pod.Namespace)
		// newPod := utils.GetUpdatedPodEnvSpec(pod, devId, nodeInfo.GetTotalGPUMemory()/nodeInfo.GetGPUCount())
		//newPod = utils.GetUpdatedPodAnnotationSpec(pod, devId, n.GetTotalGPUMemory()/n.GetGPUCount())
		patchedAnnotationBytes, err := utils.PatchPodAnnotationSpec(pod, devId, n.GetTotalGPUMemory()/n.GetGPUCount())
		if err != nil {
			return fmt.Errorf("failed to generate patched annotations,reason: %v", err)
		}
		newPod, err = clientset.CoreV1().Pods(pod.Namespace).Patch(pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes)
		//_, err = clientset.CoreV1().Pods(newPod.Namespace).Update(newPod)
		if err != nil {
			// the object has been modified; please apply your changes to the latest version and try again
			if err.Error() == OptimisticLockErrorMsg {
				// retry
				pod, err = clientset.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				// newPod = utils.GetUpdatedPodEnvSpec(pod, devId, nodeInfo.GetTotalGPUMemory()/nodeInfo.GetGPUCount())
				//newPod = utils.GetUpdatedPodAnnotationSpec(pod, devId, n.GetTotalGPUMemory()/n.GetGPUCount())
				//_, err = clientset.CoreV1().Pods(newPod.Namespace).Update(newPod)
				newPod, err = clientset.CoreV1().Pods(pod.Namespace).Patch(pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes)
				if err != nil {
					return err
				}
			} else {
				log.Printf("failed to patch pod %v", pod)
				return err
			}
		}
	} else {
		err = fmt.Errorf("The node %s can't place the pod %s in ns %s,and the pod spec is %v", pod.Spec.NodeName, pod.Name, pod.Namespace, pod)
	}

	// 2. Bind the pod to the node
	if err == nil {
		binding := &v1.Binding{
			ObjectMeta: metav1.ObjectMeta{Name: pod.Name, UID: pod.UID},
			Target:     v1.ObjectReference{Kind: "Node", Name: n.name},
		}
		log.Printf("info: Allocate() 2. Try to bind pod %s in %s namespace to node %s with %v",
			pod.Name,
			pod.Namespace,
			pod.Spec.NodeName,
			binding)
		err = clientset.CoreV1().Pods(pod.Namespace).Bind(binding)
		if err != nil {
			log.Printf("warn: Failed to bind the pod %s in ns %s due to %v", pod.Name, pod.Namespace, err)
			return err
		}
	}

	// 3. update the device info if the pod is update successfully
	if err == nil {
		log.Printf("info: Allocate() 3. Try to add pod %s in ns %s to dev %d",
			pod.Name,
			pod.Namespace,
			devId)
		dev, found := n.devs[devId]
		if !found {
			log.Printf("warn: Pod %s in ns %s failed to find the GPU ID %d in node %s", pod.Name, pod.Namespace, devId, n.name)
		} else {
			dev.addPod(newPod)
		}
	}
	log.Printf("info: Allocate() ----End to allocate GPU for gpu mem for pod %s in ns %s----", pod.Name, pod.Namespace)
	return err
}

// 计算pod的副本名称
func (n *NodeInfo) getPodReplicaName(podNs string, podName string) string {
	var buffer bytes.Buffer
	podNameItems := strings.Split(podName, "-")
	buffer.WriteString(podNs)
	for i := 0; i < len(podNameItems)-1; i++ {
		buffer.WriteString(podNameItems[i])
	}
	return buffer.String()
}


// 判断 当前GPU设备是否包含相同的副本
func (n *NodeInfo) deviceContainsSamePodReplica(deviceInfo *DeviceInfo, podNs string, podName string) bool {
	result := false
	podReplicaName := n.getPodReplicaName(podNs, podName)
	for _, podInfo := range deviceInfo.podMap {
		existsPodReplicaName := n.getPodReplicaName(podInfo.Namespace, podInfo.Name)
		if strings.Compare(podReplicaName, existsPodReplicaName) == 0 {
			result = true
			break
		}
	}
	return result
}

// 获取用户指定的GPU 设备 index
func (n *NodeInfo) getPodSpecialGPUIdx(pod *v1.Pod) (found bool, specialIdx int) {
	found = false
	specialIdx = -1
	containers := pod.Spec.Containers
	for i := 0; i < len(containers); i++ {
		container := containers[i]
		envs := container.Env
		for j := 0; j < len(envs); j++ {
			if strings.Compare(utils.EnvSpecialGPUIndex, envs[j].Name) == 0 {
				tempIdx, err := strconv.Atoi(envs[j].Value)
				if err == nil {
					specialIdx = tempIdx
					found = true
					break
				} else {
					log.Printf("[%s] env value error, will ignore [%s]", utils.EnvSpecialGPUIndex, envs[j].Value)
				}
			}
		}

	}
	return found, specialIdx
}


// allocate the GPU ID to the pod
func (n *NodeInfo) allocateGPUID(pod *v1.Pod) (candidateDevID int, found bool) {

	reqGPU := uint(0)
	found = false
	candidateDevID = -1
	availableGPUs := n.getAvailableGPUs()

	reqGPU = uint(utils.GetGPUMemoryFromPodResource(pod))

	if reqGPU > uint(0) {
		log.Printf("info: reqGPU for pod %s in ns %s: %d", pod.Name, pod.Namespace, reqGPU)
		log.Printf("info: AvailableGPUs: %v in node %s", availableGPUs, n.name)
		if len(availableGPUs) > 0 {
			var availableMemList []DeviceIndexAvailableMem  // 所有可用的设备列表
			var availableAndNotContainsReplicaList []DeviceIndexAvailableMem  // 所有可用的，并且未包含相同副本pod的

			useSpecialGPU, specialGPUIdx := n.getPodSpecialGPUIdx(pod)  // 检查是否指定了固定的gpu设备

			for devID := 0; devID < len(n.devs); devID++ {
				availableGPU, ok := availableGPUs[devID]
				if ok {
					if availableGPU >= reqGPU {  // 满足设备可用显存大小
						if useSpecialGPU {
							// 指定 gpu设备index
							if specialGPUIdx == devID {
								candidateDevID = specialGPUIdx
								found = true
								break
							}
						} else {
							// 未指定gpu设备
							// 检查当前dev设备中，是否已经调度了同一个类型的pod
							availableMemList = append(availableMemList, DeviceIndexAvailableMem{Idx: devID, AvailableMem: availableGPU})
							deviceInfo := n.devs[devID]
							if !n.deviceContainsSamePodReplica(deviceInfo, pod.Namespace, pod.Name) {
								// 未包含pod副本
								availableAndNotContainsReplicaList = append(availableAndNotContainsReplicaList, DeviceIndexAvailableMem{Idx: devID, AvailableMem: availableGPU})
							}
						}
					}
				}
			}
			// 对于未包含pod副本的设备，找出一个availableGPU最大的一个，最空闲的一个
			if !found && len(availableAndNotContainsReplicaList) > 0 {
				sort.Slice(availableAndNotContainsReplicaList, func(i, j int) bool {
					return availableAndNotContainsReplicaList[i].AvailableMem < availableAndNotContainsReplicaList[j].AvailableMem
				})
				candidateDevID = availableAndNotContainsReplicaList[len(availableAndNotContainsReplicaList)-1].Idx  // 最空闲的一个
				found = true
			}
			// 如果通过以上策略没有找到设备，则找出一个availableGPU最大的一个，最空闲的一个
			if !found && len(availableMemList) > 0 {
				sort.Slice(availableMemList, func(i, j int) bool {
					return availableMemList[i].AvailableMem < availableMemList[j].AvailableMem
				})
				candidateDevID = availableMemList[len(availableMemList)-1].Idx  // 最空闲的一个
				found = true
			}
		}

		if found {
			log.Printf("info: Find candidate dev id %d for pod %s in ns %s successfully.",
				candidateDevID,
				pod.Name,
				pod.Namespace)
		} else {
			log.Printf("warn: Failed to find available GPUs %d for the pod %s in the namespace %s",
				reqGPU,
				pod.Name,
				pod.Namespace)
		}
	}

	return candidateDevID, found
}

func (n *NodeInfo) getAvailableGPUs() (availableGPUs map[int]uint) {
	allGPUs := n.getAllGPUs()
	usedGPUs := n.getUsedGPUs()
	unhealthyGPUs := n.getUnhealthyGPUs()
	availableGPUs = map[int]uint{}
	for id, totalGPUMem := range allGPUs {
		if usedGPUMem, found := usedGPUs[id]; found {
			availableGPUs[id] = totalGPUMem - usedGPUMem
		}
	}
	log.Printf("info: available GPU list %v before removing unhealty GPUs", availableGPUs)
	for id, _ := range unhealthyGPUs {
		log.Printf("info: delete dev %d from availble GPU list", id)
		delete(availableGPUs, id)
	}
	log.Printf("info: available GPU list %v after removing unhealty GPUs", availableGPUs)

	return availableGPUs
}

// device index: gpu memory
func (n *NodeInfo) getUsedGPUs() (usedGPUs map[int]uint) {
	usedGPUs = map[int]uint{}
	for _, dev := range n.devs {
		usedGPUs[dev.idx] = dev.GetUsedGPUMemory()
	}
	log.Printf("info: getUsedGPUs: %v in node %s, and devs %v", usedGPUs, n.name, n.devs)
	return usedGPUs
}

// device index: gpu memory
func (n *NodeInfo) getAllGPUs() (allGPUs map[int]uint) {
	allGPUs = map[int]uint{}
	for _, dev := range n.devs {
		allGPUs[dev.idx] = dev.totalGPUMem
	}
	log.Printf("info: getAllGPUs: %v in node %s, and dev %v", allGPUs, n.name, n.devs)
	return allGPUs
}

// getUnhealthyGPUs get the unhealthy GPUs from configmap
func (n *NodeInfo) getUnhealthyGPUs() (unhealthyGPUs map[int]bool) {
	unhealthyGPUs = map[int]bool{}
	name := fmt.Sprintf("unhealthy-gpu-%s", n.GetName())
	log.Printf("info: try to find unhealthy node %s", name)
	cm := getConfigMap(name)
	if cm == nil {
		return
	}

	if devicesStr, found := cm.Data["gpus"]; found {
		log.Printf("warn: the unhelathy gpus %s", devicesStr)
		idsStr := strings.Split(devicesStr, ",")
		for _, sid := range idsStr {
			id, err := strconv.Atoi(sid)
			if err != nil {
				log.Printf("warn: failed to parse id %s due to %v", sid, err)
			}
			unhealthyGPUs[id] = true
		}
	} else {
		log.Println("info: skip, because there are no unhealthy gpus")
	}

	return

}
