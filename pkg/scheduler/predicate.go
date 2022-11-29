package scheduler

import (
	"fmt"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/cache"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/log"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/utils"
	"k8s.io/api/core/v1"
	schedulerapi "k8s.io/kube-scheduler/extender/v1"
)

type Predicate struct {
	Name  string
	cache *cache.SchedulerCache
}

func (p Predicate) checkNode(pod *v1.Pod, nodeName string, c *cache.SchedulerCache) (*v1.Node, error) {
	log.V(10).Info("info: check if the pod name %s can be scheduled on node %s", pod.Name, nodeName)
	nodeInfo, err := c.GetNodeInfo(nodeName)
	if err != nil {
		return nil, err
	}

	node := nodeInfo.GetNode()
	if node == nil {
		return nil, fmt.Errorf("failed get node with name %s", nodeName)
	}
	if !utils.IsGPUSharingNode(node) {
		return nil, fmt.Errorf("The node %s is not for GPU share, need skip", nodeName)
	}

	allocatable := nodeInfo.Assume(pod)
	if !allocatable {
		return nil, fmt.Errorf("Insufficient GPU Memory in one device")
	} else {
		log.V(10).Info("info: The pod %s in the namespace %s can be scheduled on %s",
			pod.Name,
			pod.Namespace,
			nodeName)
	}
	return node, nil
}

func (p Predicate) Handler(args *schedulerapi.ExtenderArgs) *schedulerapi.ExtenderFilterResult {
	if args == nil || args.Pod == nil {
		return &schedulerapi.ExtenderFilterResult{Error: fmt.Sprintf("arg or pod is nil")}
	}

	pod := args.Pod
	var nodeNames []string
	if args.NodeNames != nil {
		nodeNames = *args.NodeNames
		log.V(3).Info("extender args NodeNames is not nil, result %+v", nodeNames)
	} else if args.Nodes != nil {
		for _, n := range args.Nodes.Items {
			nodeNames = append(nodeNames, n.Name)
		}
		log.V(3).Info("extender args Nodes is not nil, names is %+v", nodeNames)
	} else {
		return &schedulerapi.ExtenderFilterResult{Error: fmt.Sprintf("cannot get node names")}
	}
	canSchedule := make([]string, 0, len(nodeNames))
	canNotSchedule := make(map[string]string)
	canScheduleNodes := &v1.NodeList{}

	for _, nodeName := range nodeNames {
		node, err := p.checkNode(pod, nodeName, p.cache)
		if err != nil {
			canNotSchedule[nodeName] = err.Error()
		} else {
			if node != nil {
				canSchedule = append(canSchedule, nodeName)
				canScheduleNodes.Items = append(canScheduleNodes.Items, *node)
			}
		}
	}

	result := schedulerapi.ExtenderFilterResult{
		NodeNames:   &canSchedule,
		Nodes:       canScheduleNodes,
		FailedNodes: canNotSchedule,
		Error:       "",
	}

	log.V(100).Info("predicate result for %s, is %+v", pod.Name, result)
	return &result
}
