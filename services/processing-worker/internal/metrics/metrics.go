package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	processedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "videos_processed_total",
		Help: "Total de vídeos processados por resultado.",
	}, []string{"status"})

	processingErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "video_processing_errors_total",
		Help: "Total de erros transitórios no processamento.",
	})

	processingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "video_processing_duration_seconds",
		Help:    "Duração do processamento de vídeo em segundos.",
		Buckets: prometheus.DefBuckets,
	})
)

func ProcessedCompleted()          { processedTotal.WithLabelValues("completed").Inc() }
func ProcessedFailed()             { processedTotal.WithLabelValues("failed").Inc() }
func ProcessingError()             { processingErrorsTotal.Inc() }
func ObserveProcessing(d time.Duration) { processingDuration.Observe(d.Seconds()) }

func Serve(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, mux)
}
