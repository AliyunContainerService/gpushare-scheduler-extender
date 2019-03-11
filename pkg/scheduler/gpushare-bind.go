package scheduler

import (
	"fmt"
	"log"

	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/cache"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	OptimisticLockErrorMsg = "the object has been modified; please apply your changes to the latest version and try again"
)

func NewGPUShareBind(clientset *kubernetes.Clientset, c *cache.SchedulerCache) *Bind {
	return &Bind{
		Name: "gpusharingbinding",
		Func: func(name string, namespace string, podUID types.UID, node string, c *cache.SchedulerCache) error {
			pod, err := getPod(name, namespace, podUID, clientset, c)
			if err != nil {
				log.Printf("warn: Failed to handle pod %s in ns %s due to error %v", name, namespace, err)
				return err
			}

			nodeInfo, err := c.GetNodeInfo(node)
			if err != nil {
				log.Printf("warn: Failed to handle pod %s in ns %s due to error %v", name, namespace, err)
				return err
			}
			err = nodeInfo.Allocate(clientset, pod)
			if err != nil {
				log.Printf("warn: Failed to handle pod %s in ns %s due to error %v", name, namespace, err)
				return err
			}
			return nil
		},
		cache: c,
	}
}

func getPod(name string, namespace string, podUID types.UID, clientset *kubernetes.Clientset, c *cache.SchedulerCache) (pod *v1.Pod, err error) {
	pod, err = c.GetPod(name, namespace)
	if errors.IsNotFound(err) {
		pod, err = clientset.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if pod.UID != podUID {
		pod, err = clientset.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if pod.UID != podUID {
			return nil, fmt.Errorf("The pod %s in ns %s's uid is %v, and it's not equal with expected %v",
				name,
				namespace,
				pod.UID,
				podUID)
		}
	}

	return pod, nil
}
