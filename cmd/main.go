package main

import (
	"context"
	"github.com/asaskevich/EventBus"
	"github.com/maxaizer/hh-parser/internal/bot"
	"github.com/maxaizer/hh-parser/internal/clients/gemini"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/config"
	"github.com/maxaizer/hh-parser/internal/logger"
	"github.com/maxaizer/hh-parser/internal/metrics"
	"github.com/maxaizer/hh-parser/internal/repositories"
	"github.com/maxaizer/hh-parser/internal/services"
	log "github.com/sirupsen/logrus"
)

func main() {

	cfg := config.Get()

	logger.Setup(cfg.Logger)
	defer logger.Cleanup()

	metrics.StartMetricsServer()

	dbContext, err := repositories.NewDbContext(cfg.DB.ConnectionString)
	if err != nil {
		log.Fatalf("can't create db context: %v", err)
	}
	err = dbContext.Migrate()
	if err != nil {
		log.Fatalf("can't migrate db context: %v", err)
	}
	searches := repositories.NewSearchRepository(dbContext.DB)
	regions := repositories.NewCachedRegions(repositories.NewRegionsRepository(dbContext.DB))
	vacancies := repositories.NewVacanciesRepository(dbContext.DB)

	aiClient, err := gemini.NewClient(context.Background(), cfg.Bot.AIKey)
	if err != nil {
		log.Fatalf("can't create AI service: %v", err)
	}
	aiClient.SetRateLimit(cfg.Bot.AiMaxRequestsPerSecond)

	bus := EventBus.New()

	tgbot, err := bot.NewBot(cfg.Bot.Token, bus, searches, regions)
	if err != nil {
		log.Fatalf("can't create bot: %v", err)
	}
	go tgbot.Start()

	hhClient := hh.NewClient()
	hhClient.SetRateLimit(cfg.Bot.HhMaxRequestsPerSecond)

	aiService := services.NewAIService(aiClient)

	cleaner, err := services.NewVacanciesCleaner(vacancies)
	if err != nil {
		log.Fatalf("can't create clearer: %v", err)
	}

	analyzer, err := services.NewVacanciesAnalyzer(bus, aiService, hhClient, searches, vacancies)
	if err != nil {
		log.Fatalf("can't create analyzer: %v", err)
	}
	analyzer.Run()

	defer cleaner.StopCron()
}
