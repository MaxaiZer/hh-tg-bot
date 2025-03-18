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

func setupLogger(cfg *config.Config) {

	var level log.Level

	switch cfg.Logger.LogLevel {
	case config.LevelInfo:
		level = log.InfoLevel
	case config.LevelDebug:
		level = log.DebugLevel
	case config.LevelWarning:
		level = log.WarnLevel
	case config.LevelError:
		level = log.ErrorLevel
	case config.LevelFatal:
		level = log.FatalLevel
	default:
		level = log.InfoLevel
	}
	logger.Setup(logger.Config{
		LogLevel:     level,
		AppName:      cfg.Logger.AppName,
		LokiURL:      cfg.Logger.LokiURL,
		LokiUser:     cfg.Logger.LokiUser,
		LokiPassword: cfg.Logger.LokiPassword,
	})
}

func runAnalyzer(ctx context.Context, cfg *config.Config, vacancies *repositories.Vacancies,
	searches *repositories.Searches, bus EventBus.Bus) {

	aiClient, err := gemini.NewClient(ctx, cfg.Bot.AIKey, cfg.Bot.AiModel)
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

	retriever := services.NewHHVacanciesRetriever(hhClient)

	analyzer, err := services.NewVacanciesAnalyzer(bus, aiService, retriever, searches, vacancies, cfg.Bot.AnalysisInterval)
	if err != nil {
		log.Fatalf("can't create analyzer: %v", err)
	}
	go analyzer.Run()
}

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Get()

	setupLogger(cfg)
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
	data := repositories.NewDataRepository(dbContext.DB)
	//ToDo: separate func to run bot
	bus := EventBus.New()

	tgbot, err := bot.NewBot(cfg.Bot.Token, bus, bot.Repositories{
		Search: searches,
		Region: regions,
		Data:   data,
	})
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
