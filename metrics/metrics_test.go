package metrics_test

import (
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/yakshaving.art/git-pull-mirror/metrics"
	"testing"
)

func TestMetricsAreRegistered(t *testing.T) {
	tt := []struct {
		name      string
		collector prometheus.Collector
	}{
		{
			"server is up",
			metrics.ServerIsUp,
		},
		{
			"repo is up",
			metrics.RepoIsUp,
		},
		{
			"latency seconds",
			metrics.GitLatencySecondsTotal,
		},
		{
			"hooks accepted",
			metrics.HooksAcceptedTotal,
		},
		{
			"hook retried",
			metrics.HooksRetriedTotal,
		},
		{
			"hooks failed",
			metrics.HooksFailedTotal,
		},
		{
			"hooks received",
			metrics.HooksReceivedTotal,
		},
		{
			"hooks updated",
			metrics.HooksUpdatedTotal,
		},
		{
			"config apply",
			metrics.LastSuccessfulConfigApply,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := prometheus.Register(tc.collector)
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				t.Fatalf("metric is not registered ")
			}

		})
	}
}
