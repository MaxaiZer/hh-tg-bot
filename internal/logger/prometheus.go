package logger

import (
	"github.com/maxaizer/hh-parser/internal/metrics"
	log "github.com/sirupsen/logrus"
)

type prometheusHook struct{}

func (h *prometheusHook) Fire(entry *log.Entry) error {
	errorType, ok := entry.Data[ErrorTypeField].(string)
	if !ok {
		errorType = "unknown"
	}

	metrics.ErrorsCounter.WithLabelValues(errorType).Inc()
	return nil
}

func (h *prometheusHook) Levels() []log.Level {
	return []log.Level{
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}
}

func addPrometheusHook() {
	log.AddHook(&prometheusHook{})
	log.Info("Prometheus logging enabled")
}
