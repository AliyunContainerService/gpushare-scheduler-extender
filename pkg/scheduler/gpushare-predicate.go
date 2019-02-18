package scheduler

import (
	"fmt"
	"log"

	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/cache"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/utils"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func NewGPUsharePredicate(clientset *kubernetes.Clientset, c *cache.SchedulerCache) *Predicate {
	return &Predicate{
		Name: "gpusharingfilter",
		Func: func(pod *v1.Pod, nodeName string, c *cache.SchedulerCache) (bool, error) {
			log.Printf("debug: check if the pod name %s can be scheduled on node %s", pod.Name, nodeName)
			nodeInfo, err := c.GetNodeInfo(nodeName)
			if err != nil {
				return false, err
			}

			if !utils.IsGPUSharingNode(nodeInfo.GetNode()) {
				return false, fmt.Errorf("The node %s is not for GPU share, need skip", nodeName)
			}

			allocatable := nodeInfo.Assume(pod)
			if !allocatable {
				return false, fmt.Errorf("Insufficient GPU Memory in one device")
			} else {
				log.Printf("debug: The pod %s in the namespace %s can be scheduled on %s",
					pod.Name,
					pod.Namespace,
					nodeName)
			}
			return true, nil
		},
		cache: c,
	}
}
