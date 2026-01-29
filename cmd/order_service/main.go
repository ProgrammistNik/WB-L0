package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"l0/internal/cache"
	"l0/internal/cache/lru_cache"
	"l0/internal/config"
	"l0/internal/db"
	"l0/internal/kafka"
	"l0/internal/models"
	"l0/internal/server"
	"l0/internal/service"
)

func main() {
	cfg, err := config.LoadConfig("config/config.yml")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.NewDBWithConfig(ctx, cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize database")
	}

	repository, err := db.NewOrderRepo(ctx, cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize repository")
	}

	lruCache, err := lru_cache.NewLRUCache[string, *models.Order](cfg.Cache.Capacity)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize LRU cache")
	}

	cacheLogger := logger.With().Str("component", "cache-manager").Logger()
	cacheManager := cache.NewManager(lruCache, repository, &cacheLogger)

	serviceLogger := logger.With().Str("component", "order-service").Logger()
	orderService := service.NewOrderService(cacheManager, &serviceLogger)

	if err := orderService.WarmCache(ctx); err != nil {
		logger.Warn().Err(err).Msg("Failed to warm cache, continuing with empty cache")
	}

	serverLogger := logger.With().Str("component", "http-server").Logger()
	httpServer := server.New(cfg, orderService, &serverLogger)

	kafkaLogger := logger.With().Str("component", "kafka-consumer").Logger()
	kafkaConsumer := kafka.NewConsumer(*cfg, orderService, &kafkaLogger)

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Start(); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := kafkaConsumer.Start(ctx); err != nil {
			errChan <- fmt.Errorf("Kafka consumer error: %w", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	select {
	case err := <-errChan:
		logger.Fatal().Err(err).Msg("Failed to start application")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		var stopWg sync.WaitGroup
		var stopErrors []error
		var mu sync.Mutex

		stopWg.Add(1)
		go func() {
			defer stopWg.Done()
			if err := kafkaConsumer.Stop(shutdownCtx); err != nil {
				mu.Lock()
				stopErrors = append(stopErrors, fmt.Errorf("failed to stop Kafka consumer: %w", err))
				mu.Unlock()
			}
		}()

		stopWg.Add(1)
		go func() {
			defer stopWg.Done()
			if err := httpServer.Stop(shutdownCtx); err != nil {
				mu.Lock()
				stopErrors = append(stopErrors, fmt.Errorf("failed to stop HTTP server: %w", err))
				mu.Unlock()
			}
		}()

		stopWg.Wait()

		database.Close()

		if len(stopErrors) > 0 {
			logger.Error().Int("error_count", len(stopErrors)).Msg("Some components failed to stop gracefully")
		}

		cancel()
	}()

	<-ctx.Done()
}