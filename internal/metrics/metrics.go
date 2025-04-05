package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var (
	ActiveSearches = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bot_active_searches",
		Help: "Current number of vacancies searches.",
	})
	ActiveUsers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bot_active_users",
		Help: "Current number of active users (with searches).",
	})
	ErrorsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_errors_total",
			Help: "Total number of occurred errors.",
		},
		[]string{"type"},
	)
	AnalysisDuration = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bot_vacancies_analysis_duration_seconds",
			Help: "Duration of each vacancies analysis in seconds.",
		},
	)
	HandledVacanciesCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bot_vacancies_analyzed_total",
			Help: "Total number of handled vacancies.",
		},
	)
	ApprovedByAiVacanciesCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bot_vacancies_ai_approved_total",
			Help: "Total number of vacancies that were approved by AI.",
		},
	)
	RejectedByAiVacanciesCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bot_vacancies_ai_rejected_total",
			Help: "Total number of vacancies that were rejected by AI.",
		},
	)
)

func StartMetricsServer() {

	prometheus.MustRegister(ActiveSearches)
	prometheus.MustRegister(ActiveUsers)
	prometheus.MustRegister(ErrorsCounter)
	prometheus.MustRegister(AnalysisDuration)
	prometheus.MustRegister(HandledVacanciesCounter)
	prometheus.MustRegister(ApprovedByAiVacanciesCounter)
	prometheus.MustRegister(RejectedByAiVacanciesCounter)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
}
