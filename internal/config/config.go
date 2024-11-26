package config

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
)

type Config struct {
	Logger LoggerConfig `mapstructure:"logger"`
	Bot    BotConfig    `mapstructure:"bot"`
	DB     DBConfig     `mapstructure:"db"`
}

var configFile = "./configs/config.yaml"

func Get() *Config {

	if value, _ := os.LookupEnv("MODE"); value == "test" {
		configFile = "../../configs/config.yaml"
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

	viper.SetDefault("PORT", 8080)
	viper.SetDefault("MODE", "release")

	err := bindEnvironmentVariables()
	if err != nil {
		return nil, err
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	config := Config{}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func bindEnvironmentVariables() error {
	var errs []error

	bot, db, logger := BotConfig{}, DBConfig{}, LoggerConfig{}

	if err := bot.bindEnvironmentVariables(); err != nil {
		errs = append(errs, fmt.Errorf("BotConfig: %w", err))
	}

	if err := db.bindEnvironmentVariables(); err != nil {
		errs = append(errs, fmt.Errorf("DBConfig: %w", err))
	}

	if err := logger.bindEnvironmentVariables(); err != nil {
		errs = append(errs, fmt.Errorf("LoggerConfig: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred: %w", errors.Join(errs...))
	}

	return nil
}

func (config Config) validate() error {
	var errs []error

	if err := config.DB.validate(); err != nil {
		errs = append(errs, fmt.Errorf("DBConfig: %w", err))
	}

	if err := config.Bot.validate(); err != nil {
		errs = append(errs, fmt.Errorf("BotConfig: %w", err))
	}

	if err := config.Logger.validate(); err != nil {
		errs = append(errs, fmt.Errorf("LoggerConfig: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred: %w", errors.Join(errs...))
	}

	return nil
}
