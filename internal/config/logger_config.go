package config

import (
	"errors"
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
	LogLevel     logLevel `mapstructure:"log_level"`
	AppName      string   `mapstructure:"app_name"`
	LokiURL      string   `mapstructure:"loki_url"`
	LokiUser     string   `mapstructure:"loki_user"`
	LokiPassword string   `mapstructure:"loki_password"`
	OutputFile   string   `mapstructure:"output_file"`
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
		return fmt.Errorf("multiple errors occurred: %w", errors.Join(errs...))
	}

	return nil
}

func (config LoggerConfig) bindEnvironmentVariables() error {

	err := viper.BindEnv("logger.loki_url", "LOKI_URL")
	if err != nil {
		return err
	}

	err = viper.BindEnv("logger.loki_user", "LOKI_USER")
	if err != nil {
		return err
	}

	err = viper.BindEnv("logger.loki_password", "LOKI_PASSWORD")
	if err != nil {
		return err
	}

	err = viper.BindEnv("logger.app_name", "APP_NAME")
	if err != nil {
		return err
	}

	return viper.BindEnv("logger.log_level", "LOG_LEVEL")
}
