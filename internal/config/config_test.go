package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test_Config_EnvironmentOverrideWorksCorrect(t *testing.T) {
	override := Config{
		Logger: LoggerConfig{
			LogLevel:     "Warning",
			LokiURL:      "http://loki:30000",
			LokiUser:     "loki",
			LokiPassword: "pass",
			OutputFile:   "logggs",
			AppName:      "TgBot",
		},
		Bot: BotConfig{
			Token:                  "overrideToken",
			AIKey:                  "overrideKey",
			HhMaxRequestsPerSecond: 99,
			AiMaxRequestsPerMinute: 88,
		},
		DB: DBConfig{
			ConnectionString: "newConnectionString",
		},
	}
	os.Setenv("MODE", "test")
	os.Setenv("AI_KEY", override.Bot.AIKey)
	os.Setenv("TOKEN", override.Bot.Token)
	os.Setenv("DB_CONNECTION_STRING", override.DB.ConnectionString)
	os.Setenv("LOKI_URL", override.Logger.LokiURL)
	os.Setenv("LOKI_USER", override.Logger.LokiUser)
	os.Setenv("LOKI_PASSWORD", override.Logger.LokiPassword)
	os.Setenv("LOG_LEVEL", string(override.Logger.LogLevel))
	os.Setenv("APP_NAME", override.Logger.AppName)

	cfg := Get()

	assert.Equal(t, override.Logger.LogLevel, cfg.Logger.LogLevel)
	assert.Equal(t, override.Logger.AppName, override.Logger.AppName)
	assert.Equal(t, override.Logger.LokiURL, cfg.Logger.LokiURL)
	assert.Equal(t, override.Logger.LokiUser, cfg.Logger.LokiUser)
	assert.Equal(t, override.Logger.LokiPassword, cfg.Logger.LokiPassword)
	assert.Equal(t, override.DB.ConnectionString, cfg.DB.ConnectionString)
	assert.Equal(t, override.Bot.AIKey, override.Bot.AIKey)
	assert.Equal(t, override.Bot.Token, override.Bot.Token)
}
