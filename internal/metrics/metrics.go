package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var (
	ErrorsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_errors_total",
			Help: "Total number of occurred errors.",
		},
		[]string{"type"},
	)
	AnalysisDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "bot_vacancies_analysis_duration_seconds",
			Help:    "Duration of each vacancies analysis in seconds.",
			Buckets: []float64{60, 300, 900, 1800, 3600},
		},
	)
	AnalysisStepDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "bot_vacancy_analysis_step_duration_seconds",
			Help:       "Duration of each step in the vacancy analysis process.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"step"},
	)
	HandledVacanciesCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bot_vacancies_analyzed_total",
			Help: "Total number of handled vacancies.",
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

	prometheus.MustRegister(ErrorsCounter)
	prometheus.MustRegister(HandledVacanciesCounter)
	prometheus.MustRegister(RejectedByAiVacanciesCounter)
	prometheus.MustRegister(AnalysisDuration)
	prometheus.MustRegister(AnalysisStepDuration)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
}
