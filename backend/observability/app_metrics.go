package observability

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	gitOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "githost_git_operations_total",
			Help: "Total Git HTTP operations",
		},
		[]string{"op", "result"},
	)

	gitTransferBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "githost_git_transfer_bytes_total",
			Help: "Total bytes transferred via Git HTTP",
		},
		[]string{"direction"},
	)

	webhookDeliveriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "githost_webhook_deliveries_total",
			Help: "Total webhook delivery attempts",
		},
		[]string{"result"},
	)

	webhookDeliveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "githost_webhook_delivery_duration_seconds",
			Help:    "Webhook delivery duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"},
	)

	dbConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "githost_db_connections",
			Help: "Current open database connections",
		},
	)

	ciJobsRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "githost_ci_jobs_running",
			Help: "Currently running CI jobs",
		},
	)

	ciJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "githost_ci_jobs_total",
			Help: "Total CI jobs completed",
		},
		[]string{"result"},
	)
)

func init() {
	if err := prometheus.Register(gitOpsTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
	if err := prometheus.Register(gitTransferBytes); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
	if err := prometheus.Register(webhookDeliveriesTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
	if err := prometheus.Register(webhookDeliveryDuration); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
	if err := prometheus.Register(dbConnections); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
	if err := prometheus.Register(ciJobsRunning); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
	if err := prometheus.Register(ciJobsTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
}

func RecordGitOperation(op, result string) {
	gitOpsTotal.WithLabelValues(op, result).Inc()
}

func AddGitTransferBytes(direction string, n float64) {
	gitTransferBytes.WithLabelValues(direction).Add(n)
}

func RecordWebhookDelivery(result string, durationSeconds float64) {
	webhookDeliveriesTotal.WithLabelValues(result).Inc()
	webhookDeliveryDuration.WithLabelValues(result).Observe(durationSeconds)
}

func SetDBConnections(n float64) {
	dbConnections.Set(n)
}

func IncCIJobsRunning() {
	ciJobsRunning.Inc()
}

func DecCIJobsRunning() {
	ciJobsRunning.Dec()
}

func RecordCIJob(result string) {
	ciJobsTotal.WithLabelValues(result).Inc()
	ciJobsRunning.Dec()
}
