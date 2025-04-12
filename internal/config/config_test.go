package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test_Config_EnvironmentOverrideWorksCorrect(t *testing.T) {
	override := Config{
		Logger: LoggerConfig{
			LogLevel:   "Warning",
			OutputFile: "logggs",
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
	os.Setenv("LOG_LEVEL", string(override.Logger.LogLevel))

	cfg := Get()

	assert.Equal(t, override.Logger.LogLevel, cfg.Logger.LogLevel)
	assert.Equal(t, override.DB.ConnectionString, cfg.DB.ConnectionString)
	assert.Equal(t, override.Bot.AIKey, override.Bot.AIKey)
	assert.Equal(t, override.Bot.Token, override.Bot.Token)
}
