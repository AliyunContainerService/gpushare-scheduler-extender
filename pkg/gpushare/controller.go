package gpushare

import (
	"fmt"
	"time"

	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/cache"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/utils"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientgocache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"log"

	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"
)

var (
	KeyFunc = clientgocache.DeletionHandlingMetaNamespaceKeyFunc
)

type Controller struct {
	clientset *kubernetes.Clientset

	// podLister can list/get pods from the shared informer's store.
	podLister corelisters.PodLister

	// nodeLister can list/get nodes from the shared informer's store.
	nodeLister corelisters.NodeLister

	// podQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	podQueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// podInformerSynced returns true if the pod store has been synced at least once.
	podInformerSynced clientgocache.InformerSynced

	// nodeInformerSynced returns true if the service store has been synced at least once.
	nodeInformerSynced clientgocache.InformerSynced

	schedulerCache *cache.SchedulerCache

	// The cache to store the pod to be removed
	removePodCache map[string]*v1.Pod
}

func NewController(clientset *kubernetes.Clientset, kubeInformerFactory kubeinformers.SharedInformerFactory, stopCh <-chan struct{}) (*Controller, error) {
	log.Printf("info: Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	// eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "gpushare-schd-extender"})

	c := &Controller{
		clientset:      clientset,
		podQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "podQueue"),
		recorder:       recorder,
		removePodCache: map[string]*v1.Pod{},
	}
	// Create pod informer.
	podInformer := kubeInformerFactory.Core().V1().Pods()
	podInformer.Informer().AddEventHandler(clientgocache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			switch t := obj.(type) {
			case *v1.Pod:
				// log.Printf("debug: added pod %s in ns %s", t.Name, t.Namespace)
				return utils.IsGPUsharingPod(t)
			case clientgocache.DeletedFinalStateUnknown:
				if pod, ok := t.Obj.(*v1.Pod); ok {
					log.Printf("debug: delete pod %s in ns %s", pod.Name, pod.Namespace)
					return utils.IsGPUsharingPod(pod)
				}
				runtime.HandleError(fmt.Errorf("unable to convert object %T to *v1.Pod in %T", obj, c))
				return false
			default:
				runtime.HandleError(fmt.Errorf("unable to handle object in %T: %T", c, obj))
				return false
			}
		},
		Handler: clientgocache.ResourceEventHandlerFuncs{
			AddFunc:    c.addPodToCache,
			UpdateFunc: c.updatePodInCache,
			DeleteFunc: c.deletePodFromCache,
		},
	})

	c.podLister = podInformer.Lister()
	c.podInformerSynced = podInformer.Informer().HasSynced

	// Create node informer
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()
	c.nodeLister = nodeInformer.Lister()
	c.nodeInformerSynced = nodeInformer.Informer().HasSynced

	// Start informer goroutines.
	go kubeInformerFactory.Start(stopCh)

	// Create scheduler Cache
	c.schedulerCache = cache.NewSchedulerCache(c.nodeLister, c.podLister)

	log.Println("info: begin to wait for cache")

	if ok := clientgocache.WaitForCacheSync(stopCh, c.nodeInformerSynced); !ok {
		return nil, fmt.Errorf("failed to wait for node caches to sync")
	} else {
		log.Println("info: init the node cache successfully")
	}

	log.Println("info: end to wait for cache")

	return c, nil
}

func (c *Controller) BuildCache() error {
	return c.schedulerCache.BuildCache()
}

func (c *Controller) GetSchedulerCache() *cache.SchedulerCache {
	return c.schedulerCache
}

// Run will set up the event handlers
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.podQueue.ShutDown()

	log.Println("info: Starting GPU Sharing Controller.")
	log.Println("info: Waiting for informer caches to sync")

	log.Printf("info: Starting %v workers.", threadiness)
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Println("info: Started workers")
	<-stopCh
	log.Println("info: Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// syncPod will sync the pod with the given key if it has had its expectations fulfilled,
// meaning it did not expect to see any more of its pods created or deleted. This function is not meant to be
// invoked concurrently with the same key.
func (c *Controller) syncPod(key string) (forget bool, err error) {
	ns, name, err := clientgocache.SplitMetaNamespaceKey(key)
	log.Printf("debug: begin to sync gpushare pod %s in ns %s", name, ns)
	if err != nil {
		return false, err
	}

	pod, err := c.podLister.Pods(ns).Get(name)
	switch {
	case errors.IsNotFound(err):
		log.Printf("debug: pod %s in ns %s has been deleted.", name, ns)
		pod, found := c.removePodCache[key]
		if found {
			c.schedulerCache.RemovePod(pod)
			delete(c.removePodCache, key)
		}
	case err != nil:
		log.Printf("warn: unable to retrieve pod %v from the store: %v", key, err)
	default:
		if utils.IsCompletePod(pod) {
			log.Printf("debug: pod %s in ns %s has completed.", name, ns)
			c.schedulerCache.RemovePod(pod)
		} else {
			err := c.schedulerCache.AddOrUpdatePod(pod)
			if err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

// processNextWorkItem will read a single work item off the podQueue and
// attempt to process it.
func (c *Controller) processNextWorkItem() bool {
	log.Println("begin processNextWorkItem()")
	key, quit := c.podQueue.Get()
	if quit {
		return false
	}
	defer c.podQueue.Done(key)
	defer log.Println("end processNextWorkItem()")
	forget, err := c.syncPod(key.(string))
	if err == nil {
		// log.Printf("Error syncing pods: %v", err)
		if forget {
			c.podQueue.Forget(key)
		}
		return false
	}

	log.Printf("Error syncing pods: %v", err)
	runtime.HandleError(fmt.Errorf("Error syncing pod: %v", err))
	c.podQueue.AddRateLimited(key)

	return true
}

func (c *Controller) addPodToCache(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		log.Printf("warn: cannot convert to *v1.Pod: %v", obj)
		return
	}

	// if !assignedNonTerminatedPod(t) {
	// 	log.Printf("debug: skip pod %s due to it's terminated.", pod.Name)
	// 	return
	// }

	podKey, err := KeyFunc(pod)
	if err != nil {
		log.Printf("warn: Failed to get the jobkey: %v", err)
		return
	}

	c.podQueue.Add(podKey)

	// NOTE: Updating equivalence cache of addPodToCache has been
	// handled optimistically in: pkg/scheduler/scheduler.go#assume()
}

func (c *Controller) updatePodInCache(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*v1.Pod)
	if !ok {
		log.Printf("warn: cannot convert oldObj to *v1.Pod: %v", oldObj)
		return
	}
	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		log.Printf("warn: cannot convert newObj to *v1.Pod: %v", newObj)
		return
	}
	needUpdate := false

	podUID := oldPod.UID

	// 1. Need update when pod is turned to complete or failed
	if c.schedulerCache.KnownPod(podUID) && utils.IsCompletePod(newPod) {
		needUpdate = true
	}
	// 2. Need update when it's unknown pod, and GPU annotation has been set
	if !c.schedulerCache.KnownPod(podUID) && utils.GetGPUIDFromAnnotation(newPod) >= 0 {
		needUpdate = true
	}
	if needUpdate {
		podKey, err := KeyFunc(newPod)
		if err != nil {
			log.Printf("warn: Failed to get the jobkey: %v", err)
			return
		}
		log.Printf("info: Need to update pod name %s in ns %s and old status is %v, new status is %v; its old annotation %v and new annotation %v",
			newPod.Name,
			newPod.Namespace,
			oldPod.Status.Phase,
			newPod.Status.Phase,
			oldPod.Annotations,
			newPod.Annotations)
		c.podQueue.Add(podKey)
	} else {
		log.Printf("debug: No need to update pod name %s in ns %s and old status is %v, new status is %v; its old annotation %v and new annotation %v",
			newPod.Name,
			newPod.Namespace,
			oldPod.Status.Phase,
			newPod.Status.Phase,
			oldPod.Annotations,
			newPod.Annotations)
	}

	return
}

func (c *Controller) deletePodFromCache(obj interface{}) {
	var pod *v1.Pod
	switch t := obj.(type) {
	case *v1.Pod:
		pod = t
	case clientgocache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*v1.Pod)
		if !ok {
			log.Printf("warn: cannot convert to *v1.Pod: %v", t.Obj)
			return
		}
	default:
		log.Printf("warn: cannot convert to *v1.Pod: %v", t)
		return
	}

	log.Printf("debug: delete pod %s in ns %s", pod.Name, pod.Namespace)
	podKey, err := KeyFunc(pod)
	if err != nil {
		log.Printf("warn: Failed to get the jobkey: %v", err)
		return
	}
	c.podQueue.Add(podKey)
	c.removePodCache[podKey] = pod
}
