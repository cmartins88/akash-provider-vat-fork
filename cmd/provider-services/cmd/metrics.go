package cmd

import (
	"net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// makeMetricsRouter builds the handler served on --metrics-listener.
//
// withPprof adds net/http/pprof under /debug/pprof. It is OFF by default: this
// process decides bids, and /debug/pprof/profile blocks for 30s of CPU
// profiling, so the debug surface is opt-in even though the listener it rides
// on is already opt-in.
func makeMetricsRouter(withPprof bool) *mux.Router {
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))

	if withPprof {
		// ⛔ PathPrefix, NOT HandleFunc("/pprof/"). gorilla/mux matches paths
		// EXACTLY, so registering "/pprof/" serves only that literal path --
		// every NAMED profile (heap, goroutine, allocs, block, mutex) 404s,
		// because those are dispatched by pprof.Index off the remaining path.
		//
		// This is not hypothetical: operator/inventory/cmd.go registers pprof
		// that way, and on a live cluster /debug/pprof/ returns 200 while
		// /debug/pprof/heap returns 404. Heap is the profile worth having.
		debug := router.PathPrefix("/debug/pprof").Subrouter()
		debug.HandleFunc("/cmdline", pprof.Cmdline)
		debug.HandleFunc("/profile", pprof.Profile)
		debug.HandleFunc("/symbol", pprof.Symbol)
		debug.HandleFunc("/trace", pprof.Trace)
		// Index also serves the named profiles from the trailing path segment.
		debug.PathPrefix("/").HandlerFunc(pprof.Index)
	}

	return router
}
