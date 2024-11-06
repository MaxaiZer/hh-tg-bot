package config

import (
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
			AiMaxRequestsPerSecond: 88,
		},
		DB: DBConfig{
			ConnectionString: "newConnectionString",
		},
	}
	os.Setenv("MODE", "test")
	os.Setenv("AI_KEY", override.Bot.AIKey)
	os.Setenv("TOKEN", override.Bot.Token)
	os.Setenv("DB_CONNECTION_STRING", override.DB.ConnectionString)

	cfg := Get()
	if cfg.Bot.AIKey != override.Bot.AIKey {
		t.Errorf("Expected AIKey to be overridden")
	}
	if cfg.DB.ConnectionString != override.DB.ConnectionString {
		t.Errorf("Expected DB.ConnectionString to be overridden")
	}
	if cfg.Bot.Token != override.Bot.Token {
		t.Errorf("Expected Bot.Token to be overridden")
	}
}
