package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var namespace = "github"
var subsystem = "webhooks"

// Prometheus metrics
var (
	ServerIsUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "up",
		Help:      "whether the service is ready to receive requests or not",
	})
	RepoIsUp = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "repo_up",
		Help:      "whether a repo is succeeding or failing to read or write",
	}, []string{"repo"})
	HooksReceivedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "hooks_received_total",
		Help:      "total number of hooks received",
	})
	HooksRetriedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "hooks_retried_total",
		Help:      "total number of hooks that failed and were retried",
	}, []string{"repo"})
	HooksAcceptedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "hooks_accepted_total",
		Help:      "number of hooks accepted",
	}, []string{"origin"})
	HooksUpdatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "hooks_updated_total",
		Help:      "number of hooks updated",
	}, []string{"repo"})
	HooksFailedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "hooks_failed_total",
		Help:      "number of hooks failed",
	}, []string{"repo"})
	GitLatencySecondsTotal = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "git_latency_seconds",
		Help:      "latency of git operations",
	}, []string{"operation", "repo"})

	bootTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "boot_time_seconds",
		Help:      "unix timestamp of when the service was started",
	})
	LastSuccessfulConfigApply = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "last_successful_config_apply",
		Help:      "unix timestamp of when the last configuration was successfully applied",
	})
)

// Register registers the metrics on the given path and the given http server
func Register(path string, server *http.ServeMux) {
	bootTime.Set(float64(time.Now().Unix()))
	ServerIsUp.Set(0)

	prometheus.MustRegister(bootTime)
	prometheus.MustRegister(LastSuccessfulConfigApply)
	prometheus.MustRegister(HooksReceivedTotal)
	prometheus.MustRegister(HooksAcceptedTotal)
	prometheus.MustRegister(HooksUpdatedTotal)
	prometheus.MustRegister(HooksFailedTotal)
	prometheus.MustRegister(GitLatencySecondsTotal)
	prometheus.MustRegister(RepoIsUp)
	prometheus.MustRegister(ServerIsUp)
	prometheus.MustRegister(HooksRetriedTotal)

	server.Handle(path, prometheus.Handler())
}
