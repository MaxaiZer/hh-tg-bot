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
	"os/signal"
	"syscall"
)

func runAnalyzer(ctx context.Context, cfg *config.Config, vacancies *repositories.Vacancies,
	searches *repositories.Searches, bus EventBus.Bus) {

	aiClient, err := gemini.NewClient(ctx, cfg.Bot.AIKey, gemini.Model15Flash)
	if err != nil {
		log.Fatalf("can't create AI service: %v", err)
	}
	aiClient.SetMinuteRateLimit(cfg.Bot.AiMaxRequestsPerMinute)
	aiClient.SetDayRateLimit(cfg.Bot.AiMaxRequestsPerDay)

	hhClient := hh.NewClient()
	hhClient.SetRateLimit(cfg.Bot.HhMaxRequestsPerSecond)

	aiService := services.NewAIService(aiClient)

	cleaner, err := services.NewVacanciesCleaner(vacancies)
	if err != nil {
		log.Fatalf("can't create clearer: %v", err)
	}
	defer cleaner.StopCron()

	analyzer, err := services.NewVacanciesAnalyzer(bus, aiService, hhClient, searches, vacancies, cfg.Bot.AnalysisInterval)
	if err != nil {
		log.Fatalf("can't create analyzer: %v", err)
	}
	go analyzer.Run()
}

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Get()

	logger.Setup(cfg.Logger)
	defer logger.Cleanup()

	metrics.StartMetricsServer()

	dbContext, err := repositories.NewDbContext(cfg.DB.ConnectionString)
	if err != nil {
		log.Fatalf("can't create db context: %v", err)
	}
	defer dbContext.Close()

	err = dbContext.Migrate()
	if err != nil {
		log.Fatalf("can't migrate db context: %v", err)
	}

	searches := repositories.NewSearchRepository(dbContext.DB)
	regions := repositories.NewCachedRegions(repositories.NewRegionsRepository(dbContext.DB))
	vacancies := repositories.NewVacanciesRepository(dbContext.DB)
	//ToDo: separate func to run bot
	bus := EventBus.New()

	tgbot, err := bot.NewBot(cfg.Bot.Token, bus, searches, regions)
	if err != nil {
		log.Fatalf("can't create bot: %v", err)
	}
	go tgbot.Run()

	runAnalyzer(ctx, cfg, vacancies, searches, bus)

	<-ctx.Done()

	log.Info("Shutting down services...")
	tgbot.Stop()
	log.Info("Services stopped.")
}
