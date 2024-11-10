package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

type BotConfig struct {
	Token                  string  `mapstructure:"token"`
	AIKey                  string  `mapstructure:"ai_key"`
	HhMaxRequestsPerSecond float32 `mapstructure:"hh_max_requests_per_second"`
	AiMaxRequestsPerMinute float32 `mapstructure:"ai_max_requests_per_minute"`
	AiMaxRequestsPerDay    float32 `mapstructure:"ai_max_requests_per_day"`
}

func (config BotConfig) validate() error {

	var missingFields []string

	if config.Token == "" {
		missingFields = append(missingFields, "token")
	}

	if config.AIKey == "" {
		missingFields = append(missingFields, "ai_key")
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
		return createMultiError(errs)
	}

	return nil
}
