package cache

import (
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	corelisters "k8s.io/client-go/listers/core/v1"
	clientgocache "k8s.io/client-go/tools/cache"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ConfigMapLister         corelisters.ConfigMapLister
	ConfigMapInformerSynced clientgocache.InformerSynced
)

func getConfigMap(name string) *v1.ConfigMap {
	configMap, err := ConfigMapLister.ConfigMaps(metav1.NamespaceSystem).Get(name)

	// If we can't get the configmap just return nil. The resync will eventually
	// sync things up.
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.V(10).Info("warn: find configmap with error: %v", err)
			utilruntime.HandleError(err)
		}
		return nil
	}

	return configMap
}
