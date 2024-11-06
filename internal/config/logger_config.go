package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type logLevel string

const (
	LevelInfo    logLevel = "INFO"
	LevelDebug   logLevel = "DEBUG"
	LevelWarning logLevel = "WARNING"
	LevelError   logLevel = "ERROR"
	LevelFatal   logLevel = "FATAL"
)

type LoggerConfig struct {
	LogLevel   logLevel `mapstructure:"log_level"`
	OutputFile string   `mapstructure:"output_file"`
}

func (config LoggerConfig) validate() error {
	var errs []error

	if config.LogLevel == "" {
		errs = append(errs, fmt.Errorf("missing variable: log_level"))
	}
	if config.OutputFile == "" {
		errs = append(errs, fmt.Errorf("missing variable: output_file"))
	}

	if len(errs) > 0 {
		return createMultiError(errs)
	}

	return nil
}

func (config LoggerConfig) bindEnvironmentVariables() error {
	return viper.BindEnv("logger.log_level", "LOG_LEVEL")
}
