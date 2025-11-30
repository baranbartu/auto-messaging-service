package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"automessaging/internal/config"
	dbpkg "automessaging/internal/db"
	httpserver "automessaging/internal/http"
	"automessaging/internal/http/handler"
	"automessaging/internal/repository/postgres"
	"automessaging/internal/scheduler"
	"automessaging/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.Webhook.URL == "" {
		log.Fatal("WEBHOOK_URL environment variable must be set")
	}

	database, err := dbpkg.Connect(cfg.Postgres)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer database.Close()

	if err := dbpkg.RunMigrations(ctx, database, "migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	repo := postgres.NewMessageRepository(database)

	messageService := service.NewMessageService(service.Dependencies{
		Repo:  repo,
		Redis: redisClient,
	}, service.MessageServiceOptions{
		FetchLimit:     cfg.Scheduler.FetchLimit,
		WebhookURL:     cfg.Webhook.URL,
		WebhookAuthKey: cfg.Webhook.AuthKey,
	})

	schedLogger := log.New(os.Stdout, "scheduler ", log.LstdFlags)
	sched := scheduler.New(messageService, cfg.Scheduler.Interval, schedLogger)

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(appCtx); err != nil {
		log.Fatalf("start scheduler: %v", err)
	}

	controlHandler := handler.NewControlHandler(sched)
	messageHandler := handler.NewMessageHandler(messageService)
	router := httpserver.NewRouter(controlHandler, messageHandler)

	server := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	if err := sched.Stop(); err != nil && err != scheduler.ErrNotRunning {
		log.Printf("scheduler stop error: %v", err)
	}
}
