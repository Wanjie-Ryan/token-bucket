package ratelimiter

import "github.com/prometheus/client_golang/prometheus"

var checksTotal = prometheus.NewCounterVec(

	prometheus.CounterOpts{
		Name: "rate_limiter_checks_total",
		Help: "Total rate limi checks, labeled by which limiter handled it and the result",
	},
	[]string{"limiter", "result"},
)

func init() {
	prometheus.MustRegister(checksTotal)
}
