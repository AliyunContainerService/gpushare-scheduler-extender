package routes

import (
	"net/http"
	"net/http/pprof"

	"github.com/julienschmidt/httprouter"
)

func AddPProf(r *httprouter.Router) {
	r.GET("/debug/pprof/", index)
	r.GET("/debug/pprof/cmdline/", cmdline)
	r.GET("/debug/pprof/profile/", profile)
	r.GET("/debug/pprof/symbol/", symbol)
	r.GET("/debug/pprof/trace/", trace)

	r.GET("/debug/pprof/heap/", heap)
	r.GET("/debug/pprof/goroutine/", goroutine)
	r.GET("/debug/pprof/block/", block)
	r.GET("/debug/pprof/threadcreate/", threadcreate)
	r.GET("/debug/pprof/mutex/", mutex)
}

// profiling tools handlers

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Index(w, r)
}

func cmdline(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Cmdline(w, r)
}

func profile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Profile(w, r)
}

func symbol(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Symbol(w, r)
}

func trace(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Trace(w, r)
}

func heap(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Handler("heap").ServeHTTP(w, r)
}

func goroutine(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Handler("goroutine").ServeHTTP(w, r)
}

func block(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Handler("block").ServeHTTP(w, r)
}

func threadcreate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Handler("threadcreate").ServeHTTP(w, r)
}

func mutex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pprof.Handler("mutex").ServeHTTP(w, r)
}
