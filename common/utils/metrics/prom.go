package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Recorder struct {
	connectionCountGauge            prometheus.Gauge
	outgoingHttpRequestDurHistogram *prometheus.HistogramVec
	outgoingHttpRequestsInFlight    *prometheus.GaugeVec
	cacheHitCounter                 *prometheus.CounterVec
	cacheMissCounter                *prometheus.CounterVec
	anomalyCounter                  *prometheus.CounterVec
}

var HTTPClientRecorder Recorder

func (*Recorder) LogDuration(startTime time.Time, service string, handler string, method string, code int) {
	var amendedCode string
	if code >= 200 && code <= 299 {
		amendedCode = "2xx"
	} else if code >= 300 && code <= 399 {
		amendedCode = "3xx"
	} else if code >= 400 && code <= 499 {
		amendedCode = "4xx"
	} else if code >= 500 && code <= 599 {
		amendedCode = "5xx"
	} else {
		amendedCode = "n/a"
	}
	HTTPClientRecorder.outgoingHttpRequestDurHistogram.With(prometheus.Labels{
		"service": service,
		"handler": handler,
		"method":  method,
		"code":    amendedCode,
	}).Observe(time.Since(startTime).Seconds())
}

func (*Recorder) IncInFlight(service string, handler string) {
	HTTPClientRecorder.outgoingHttpRequestsInFlight.With(prometheus.Labels{
		"service": service,
		"handler": handler,
	}).Inc()
}

func (*Recorder) DecInFlight(service string, handler string) {
	HTTPClientRecorder.outgoingHttpRequestsInFlight.With(prometheus.Labels{
		"service": service,
		"handler": handler,
	}).Dec()
}

func (*Recorder) SetConnectionCount(count int) {
	HTTPClientRecorder.connectionCountGauge.Set(float64(count))
}

func (*Recorder) CountCacheHit(service string, handler string) {
	HTTPClientRecorder.cacheHitCounter.With(prometheus.Labels{
		"service": service,
		"handler": handler,
	}).Inc()
}

func (*Recorder) CountCacheMiss(service string, handler string) {
	HTTPClientRecorder.cacheMissCounter.With(prometheus.Labels{
		"service": service,
		"handler": handler,
	}).Inc()
}

func (*Recorder) CountAnomaly(anomaly string) {
	HTTPClientRecorder.anomalyCounter.With(prometheus.Labels{
		"anomaly": anomaly,
	}).Inc()
}

func init() {
	// These are modeled after the incoming prometheus metrics
	// @see: https://github.com/slok/go-http-metrics/blob/master/metrics/prometheus/prometheus.go
	HTTPClientRecorder = Recorder{
		connectionCountGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "incoming",
			Subsystem: "http",
			Name:      "connection_count_total",
			Help:      "The total connection count of incoming HTTP requests",
		}),
		outgoingHttpRequestDurHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "outgoing",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "The latency of outgoing (client-based) HTTP requests.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"service", "handler", "method", "code"}),
		outgoingHttpRequestsInFlight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "outgoing",
			Subsystem: "http",
			Name:      "requests_inflight",
			Help:      "The number of inflight outgoing requests being handled at the same time",
		}, []string{"service", "handler"}),
		cacheHitCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "cache",
			Subsystem: "redis",
			Name:      "hit",
			Help:      "The number of redis cache hits",
		}, []string{"service", "handler"}),
		cacheMissCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "cache",
			Subsystem: "redis",
			Name:      "miss",
			Help:      "The number of redis cache misses",
		}, []string{"service", "handler"}),
		anomalyCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "anomaly",
			Help: "The number of anomalies/errors",
		}, []string{"anomaly"}),
	}

	prometheus.DefaultRegisterer.MustRegister(
		HTTPClientRecorder.connectionCountGauge,
		HTTPClientRecorder.outgoingHttpRequestDurHistogram,
		HTTPClientRecorder.outgoingHttpRequestsInFlight,
		HTTPClientRecorder.cacheHitCounter,
		HTTPClientRecorder.cacheMissCounter,
		HTTPClientRecorder.anomalyCounter,
	)
}
