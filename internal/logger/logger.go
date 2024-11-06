package logger

import (
	"github.com/maxaizer/hh-parser/internal/config"
	"github.com/maxaizer/hh-parser/internal/metrics"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

const ErrorTypeField = "error_type"

const (
	ErrorTypeDb    = "db"
	ErrorTypeAiApi = "ai_api"
	ErrorTypeHhApi = "hh_api"
	ErrorTypeTgApi = "tg_api"
)

var logFile *os.File

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

func Setup(cfg config.LoggerConfig) {

	logDir := "./logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logFile, err := os.OpenFile(logDir+"/errors.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	customFormatter := &log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.000 -0700",
	}
	log.SetFormatter(customFormatter)
	log.AddHook(&prometheusHook{})

	switch cfg.LogLevel {
	case config.LevelInfo:
		log.SetLevel(log.InfoLevel)
	case config.LevelDebug:
		log.SetLevel(log.DebugLevel)
	case config.LevelWarning:
		log.SetLevel(log.WarnLevel)
	case config.LevelError:
		log.SetLevel(log.ErrorLevel)
	case config.LevelFatal:
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func Cleanup() {
	if logFile != nil {
		_ = logFile.Close()
	}
}
