package main

import (
	"context"
	"flag"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/gpushare"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/routes"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/scheduler"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/utils/signals"
	"github.com/julienschmidt/httprouter"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const RecommendedKubeConfigPathEnv = "KUBECONFIG"

var (
	clientset    *kubernetes.Clientset
	resyncPeriod = 30 * time.Second
	clientConfig clientcmd.ClientConfig
)

func initKubeClient() {
	kubeConfig := ""
	if len(os.Getenv(RecommendedKubeConfigPathEnv)) > 0 {
		// use the current context in kubeconfig
		// This is very useful for running locally.
		kubeConfig = os.Getenv(RecommendedKubeConfigPathEnv)
	}

	// Get kubernetes config.
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		log.Fatal("Error building kubeconfig: %s", err.Error())
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatal("fatal: Failed to init rest config due to %v", err)
	}
}

func main() {

	// Call Parse() to avoid noisy logs
	flag.CommandLine.Parse([]string{})
	ctx := context.Background()

	var logLevel int32 = 10
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		logLevel = 101
	case "info":
		logLevel = 50
	case "warn":
		logLevel = 10
	case "error":
		logLevel = 5
	}
	log.NewLoggerWithLevel(logLevel)

	threadness := StringToInt(os.Getenv("THREADNESS"))

	initKubeClient()
	port := os.Getenv("PORT")
	if _, err := strconv.Atoi(port); err != nil {
		port = "39999"
	}

	// Set up signals so we handle the first shutdown signal gracefully.
	stopCh := signals.SetupSignalHandler()

	informerFactory := kubeinformers.NewSharedInformerFactory(clientset, resyncPeriod)
	controller, err := gpushare.NewController(clientset, informerFactory, stopCh)
	if err != nil {
		log.Fatal("Failed to start due to %v", err)
	}
	err = controller.BuildCache()
	if err != nil {
		log.Fatal("Failed to start due to %v", err)
	}

	go controller.Run(threadness, stopCh)

	gpusharePredicate := scheduler.NewGPUsharePredicate(clientset, controller.GetSchedulerCache())
	gpushareBind := scheduler.NewGPUShareBind(ctx, clientset, controller.GetSchedulerCache())
	gpushareInspect := scheduler.NewGPUShareInspect(controller.GetSchedulerCache())

	router := httprouter.New()

	routes.AddPProf(router)
	routes.AddVersion(router)
	routes.AddPredicate(router, gpusharePredicate)
	routes.AddBind(router, gpushareBind)
	routes.AddInspect(router, gpushareInspect)

	log.V(3).Info("server starting on the port :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal("server listen fail %+v", err)
	}
}

func StringToInt(sThread string) int {
	thread := runtime.NumCPU()
	if threadInt, err := strconv.Atoi(sThread); err == nil {
		thread = threadInt
	}
	return thread
}
