package config

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"testing"
	"time"
)

func Test_Config_EnvironmentOverrideWorksCorrect(t *testing.T) {
	override := Config{
		Env:                     Production,
		TgToken:                 "overrideToken",
		AIKey:                   "overrideKey",
		AnalysisInterval:        3 * time.Hour,
		VacancyExpirationInDays: 128,
		HhMaxRequestsPerSecond:  99,
		AiModel:                 "super_duper_model",
		AiMaxRequestsPerMinute:  88,
		AiMaxRequestsPerDay:     89,
		DbConnectionString:      "newConnectionString",
	}
	os.Setenv("CONFIG_PATH", "../../configs/config.yaml")

	os.Setenv("ENV", string(override.Env))
	os.Setenv("TG_TOKEN", override.TgToken)
	os.Setenv("AI_KEY", override.AIKey)
	os.Setenv("ANALYSIS_INTERVAL", "3h")
	os.Setenv("VACANCY_EXPIRATION_DAYS", strconv.Itoa(override.VacancyExpirationInDays))
	os.Setenv("HH_MAX_REQUESTS_PER_SECOND", fmt.Sprintf("%f", override.HhMaxRequestsPerSecond))
	os.Setenv("AI_MODEL", override.AiModel)
	os.Setenv("AI_MAX_REQUESTS_PER_MINUTE", fmt.Sprintf("%f", override.AiMaxRequestsPerMinute))
	os.Setenv("AI_MAX_REQUESTS_PER_DAY", fmt.Sprintf("%f", override.AiMaxRequestsPerDay))
	os.Setenv("DB_CONNECTION_STRING", override.DbConnectionString)

	cfg := Get()

	assert.Equal(t, override.Env, cfg.Env)
	assert.Equal(t, override.TgToken, cfg.TgToken)
	assert.Equal(t, override.AIKey, cfg.AIKey)
	assert.Equal(t, override.AnalysisInterval, cfg.AnalysisInterval)
	assert.Equal(t, override.VacancyExpirationInDays, cfg.VacancyExpirationInDays)
	assert.Equal(t, override.HhMaxRequestsPerSecond, cfg.HhMaxRequestsPerSecond)
	assert.Equal(t, override.AiModel, cfg.AiModel)
	assert.Equal(t, override.AiMaxRequestsPerMinute, cfg.AiMaxRequestsPerMinute)
	assert.Equal(t, override.AiMaxRequestsPerDay, cfg.AiMaxRequestsPerDay)
	assert.Equal(t, override.DbConnectionString, cfg.DbConnectionString)
}
