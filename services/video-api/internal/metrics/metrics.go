package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total de requisições HTTP.",
	}, []string{"method", "path", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duração das requisições HTTP em segundos.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	videosUploadedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "videos_uploaded_total",
		Help: "Total de uploads de vídeo.",
	})

	videosDownloadedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "videos_downloaded_total",
		Help: "Total de downloads de vídeo.",
	})
)

func VideosUploadedInc()   { videosUploadedTotal.Inc() }
func VideosDownloadedInc() { videosDownloadedTotal.Inc() }

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		httpRequests.WithLabelValues(c.Request.Method, path, strconv.Itoa(c.Writer.Status())).Inc()
		httpDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}

func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) { h.ServeHTTP(c.Writer, c.Request) }
}
