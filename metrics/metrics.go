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
	HooksReceivedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "hooks_received_total",
		Help:      "total number of hooks received",
	})
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

func init() {
	bootTime.Set(float64(time.Now().Unix()))
	prometheus.MustRegister(bootTime)
	prometheus.MustRegister(LastSuccessfulConfigApply)
	prometheus.MustRegister(HooksReceivedTotal)
	prometheus.MustRegister(HooksAcceptedTotal)
	prometheus.MustRegister(HooksUpdatedTotal)
	prometheus.MustRegister(HooksFailedTotal)
	prometheus.MustRegister(GitLatencySecondsTotal)

	http.Handle("/metrics", prometheus.Handler())
}
