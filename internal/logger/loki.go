package logger

import (
	"context"
	"github.com/maxaizer/hh-parser/pkg/loki"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
)

type logrusAdapter struct {
}

func (l *logrusAdapter) Error(msg string, args ...any) {
	log.WithFields(log.Fields{"args": args, "source": "loki"}).Error(msg)
}

type lokiHook struct {
	pusher   *loki.Pusher
	minLevel log.Level
}

func (h *lokiHook) Fire(entry *log.Entry) error {
	//	errorType, ok := entry.Data[ErrorTypeField].(string)
	//	if !ok {
	//		errorType = "unknown"
	//	}

	if entry.Data["source"] == "loki" {
		return nil
	}

	caller := ""
	if entry.Caller != nil {
		caller = filepath.Base(entry.Caller.Function) + ":" + strconv.Itoa(entry.Caller.Line)
	}

	return h.pusher.Push(loki.LogEntry{
		Level:   entry.Level.String(),
		Message: entry.Message,
		Caller:  caller,
	})
}

func (h *lokiHook) Levels() []log.Level {
	var levels []log.Level
	for _, level := range log.AllLevels {
		if level <= h.minLevel {
			levels = append(levels, level)
		}
	}
	return levels
}

func addLokiHook(ctx context.Context, cfg loki.Config, minLevel log.Level) error {
	pusher, err := loki.New(ctx, cfg, &logrusAdapter{})
	if err != nil {
		return err
	}
	log.AddHook(&lokiHook{pusher: pusher, minLevel: minLevel})
	log.Info("Loki logging enabled")
	return nil
}
