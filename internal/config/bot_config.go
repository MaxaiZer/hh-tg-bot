package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"time"
)

type BotConfig struct {
	Token                  string        `mapstructure:"token"`
	AIKey                  string        `mapstructure:"ai_key"`
	AnalysisInterval       time.Duration `mapstructure:"analysis_interval"`
	HhMaxRequestsPerSecond float32       `mapstructure:"hh_max_requests_per_second"`
	AiModel                string        `mapstructure:"ai_model"`
	AiMaxRequestsPerMinute float32       `mapstructure:"ai_max_requests_per_minute"`
	AiMaxRequestsPerDay    float32       `mapstructure:"ai_max_requests_per_day"`
}

func (config BotConfig) validate() error {

	var missingFields []string

	if config.Token == "" {
		missingFields = append(missingFields, "token")
	}

	if config.AIKey == "" {
		missingFields = append(missingFields, "ai_key")
	}

	if config.AnalysisInterval == time.Duration(0) {
		missingFields = append(missingFields, "analysis_interval")
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required variables: %s", strings.Join(missingFields, ", "))
	}

	return nil
}

func (config BotConfig) bindEnvironmentVariables() error {
	var errs []error
	if err := viper.BindEnv("bot.ai_key", "AI_KEY"); err != nil {
		errs = append(errs, err)
	}

	if err := viper.BindEnv("bot.token", "TOKEN"); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred: %w", errors.Join(errs...))
	}

	return nil
}
