package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type DBConfig struct {
	ConnectionString string `mapstructure:"connection_string"`
}

func (config DBConfig) validate() error {
	if config.ConnectionString == "" {
		return fmt.Errorf("missing variable: db connection string")
	}
	return nil
}

func (config DBConfig) bindEnvironmentVariables() error {
	return viper.BindEnv("db.connection_string", "DB_CONNECTION_STRING")
}
