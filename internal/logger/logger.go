package logger

import (
	"context"
	"fmt"
	"github.com/maxaizer/hh-parser/pkg/loki"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const ErrorTypeField = "error_type"

const (
	ErrorTypeDb    = "db"
	ErrorTypeAiApi = "ai_api"
	ErrorTypeHhApi = "hh_api"
	ErrorTypeTgApi = "tg_api"
)

var logFile *os.File
var ctx context.Context
var cancelFunc context.CancelFunc

type Config struct {
	LogLevel     log.Level
	AppName      string
	LokiURL      string
	LokiUser     string
	LokiPassword string
}

func Setup(cfg Config) {

	ctx, cancelFunc = context.WithCancel(context.Background())
	logDir := "./logs"

	var err error
	if err = os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logFile, err = os.OpenFile(logDir+"/errors.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	customFormatter := &log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.000 -0700",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			funcName := filepath.Base(f.Function)
			fileInfo := fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
			return funcName, fileInfo
		},
	}
	log.SetFormatter(customFormatter)
	log.SetReportCaller(true)

	addPrometheusHook()
	err = addLokiHook(ctx, loki.Config{
		Url:          cfg.LokiURL,
		Username:     cfg.LokiUser,
		Password:     cfg.LokiPassword,
		BatchMaxSize: 1000,
		BatchMaxWait: 10 * time.Second,
		Labels:       map[string]string{"app": cfg.AppName},
	}, cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to add loki hook: %v", err)
	}

	log.SetLevel(cfg.LogLevel)
}

func Cleanup() {
	if logFile != nil {
		_ = logFile.Close()
	}
	cancelFunc()
}
