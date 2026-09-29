package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	emailsSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "emails_sent_total",
		Help: "Total de e-mails enviados.",
	})

	emailsFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "emails_failed_total",
		Help: "Total de falhas no envio de e-mail.",
	})
)

func EmailsSentInc()   { emailsSentTotal.Inc() }
func EmailsFailedInc() { emailsFailedTotal.Inc() }

func Serve(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, mux)
}
