package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/log"
	"io"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/AliyunContainerService/gpushare-scheduler-extender/pkg/scheduler"

	schedulerapi "k8s.io/kube-scheduler/extender/v1"
)

const (
	versionPath       = "/version"
	apiPrefix         = "/gpushare-scheduler"
	bindPrefix        = apiPrefix + "/bind"
	predicatesPrefix  = apiPrefix + "/filter"
	inspectPrefix     = apiPrefix + "/inspect/:nodename"
	inspectListPrefix = apiPrefix + "/inspect"
)

var (
	version = "0.1.0"
	// mu      sync.RWMutex
)

func checkBody(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
}

func InspectRoute(inspect *scheduler.Inspect) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		result := inspect.Handler(ps.ByName("nodename"))

		if resultBody, err := json.Marshal(result); err != nil {
			// panic(err)
			log.V(3).Info("warn: Failed due to %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			errMsg := fmt.Sprintf("{'error':'%s'}", err.Error())
			w.Write([]byte(errMsg))
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(resultBody)
		}
	}
}

func PredicateRoute(predicate *scheduler.Predicate) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		checkBody(w, r)

		// mu.RLock()
		// defer mu.RUnlock()

		var buf bytes.Buffer
		body := io.TeeReader(r.Body, &buf)

		var extenderArgs schedulerapi.ExtenderArgs
		var extenderFilterResult *schedulerapi.ExtenderFilterResult

		if err := json.NewDecoder(body).Decode(&extenderArgs); err != nil {
			log.V(3).Info("warn: failed to parse request due to error %v", err)
			extenderFilterResult = &schedulerapi.ExtenderFilterResult{
				Nodes:       nil,
				FailedNodes: nil,
				Error:       err.Error(),
			}
		} else {
			log.V(90).Info("debug: gpusharingfilter ExtenderArgs =%v", extenderArgs)
			extenderFilterResult = predicate.Handler(&extenderArgs)
		}

		if resultBody, err := json.Marshal(extenderFilterResult); err != nil {
			// panic(err)
			log.V(3).Info("warn: Failed due to %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			errMsg := fmt.Sprintf("{'error':'%s'}", err.Error())
			w.Write([]byte(errMsg))
		} else {
			log.V(100).Info("predicate: %s,  extenderFilterResult = %s ", predicate.Name, resultBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(resultBody)
		}
	}
}

func BindRoute(bind *scheduler.Bind) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		checkBody(w, r)

		// mu.Lock()
		// defer mu.Unlock()
		var buf bytes.Buffer
		body := io.TeeReader(r.Body, &buf)

		var extenderBindingArgs schedulerapi.ExtenderBindingArgs
		var extenderBindingResult *schedulerapi.ExtenderBindingResult
		failed := false

		if err := json.NewDecoder(body).Decode(&extenderBindingArgs); err != nil {
			extenderBindingResult = &schedulerapi.ExtenderBindingResult{
				Error: err.Error(),
			}
			failed = true
		} else {
			log.V(10).Info("debug: gpusharingBind ExtenderArgs =%v", extenderBindingArgs)
			extenderBindingResult = bind.Handler(extenderBindingArgs)
		}

		if len(extenderBindingResult.Error) > 0 {
			failed = true
		}

		if resultBody, err := json.Marshal(extenderBindingResult); err != nil {
			log.V(3).Info("warn: Failed due to %v", err)
			// panic(err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			errMsg := fmt.Sprintf("{'error':'%s'}", err.Error())
			w.Write([]byte(errMsg))
		} else {
			log.V(3).Info("info: extenderBindingResult = ", string(resultBody))
			w.Header().Set("Content-Type", "application/json")
			if failed {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}

			w.Write(resultBody)
		}
	}
}

func VersionRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, fmt.Sprint(version))
}

func AddVersion(router *httprouter.Router) {
	router.GET(versionPath, DebugLogging(VersionRoute, versionPath))
}

func DebugLogging(h httprouter.Handle, path string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		log.V(90).Info("path: %s, request body = %s", path, r.Body)
		startTime := time.Now()
		h(w, r, p)
		log.V(90).Info("path: %s, response: %v, cost_time: %v", path, w, time.Now().Sub(startTime))
	}
}

func AddPredicate(router *httprouter.Router, predicate *scheduler.Predicate) {
	// path := predicatesPrefix + "/" + predicate.Name
	router.POST(predicatesPrefix, DebugLogging(PredicateRoute(predicate), predicatesPrefix))
}

func AddBind(router *httprouter.Router, bind *scheduler.Bind) {
	if handle, _, _ := router.Lookup("POST", bindPrefix); handle != nil {
		log.V(3).Info("warning: AddBind was called more then once!")
	} else {
		router.POST(bindPrefix, DebugLogging(BindRoute(bind), bindPrefix))
	}
}

func AddInspect(router *httprouter.Router, inspect *scheduler.Inspect) {
	router.GET(inspectPrefix, DebugLogging(InspectRoute(inspect), inspectPrefix))
	router.GET(inspectListPrefix, DebugLogging(InspectRoute(inspect), inspectListPrefix))
}
