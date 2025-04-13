package config

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"time"
)

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

type Config struct {
	Env                     Environment   `mapstructure:"env"`
	TgToken                 string        `mapstructure:"tg_token" validate:"required"`
	AIKey                   string        `mapstructure:"ai_key" validate:"required"`
	AnalysisInterval        time.Duration `mapstructure:"analysis_interval" validate:"required"`
	VacancyExpirationInDays int           `mapstructure:"vacancy_expiration_days" validate:"required"`
	HhMaxRequestsPerSecond  float32       `mapstructure:"hh_max_requests_per_second" validate:"required"`
	AiModel                 string        `mapstructure:"ai_model" validate:"required"`
	AiMaxRequestsPerMinute  float32       `mapstructure:"ai_max_requests_per_minute" validate:"required"`
	AiMaxRequestsPerDay     float32       `mapstructure:"ai_max_requests_per_day" validate:"required"`
	DbConnectionString      string        `mapstructure:"db_connection_string" validate:"required"`
}

var configFile = "./configs/config.yaml"

func Get() *Config {

	if path, exists := os.LookupEnv("CONFIG_PATH"); exists {
		configFile = path
	}

	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func loadConfig(file string) (*Config, error) {

	viper.SetConfigFile(file)
	viper.AutomaticEnv()
	viper.SetDefault("env", string(Development))

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	config := Config{}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return &config, nil
}
