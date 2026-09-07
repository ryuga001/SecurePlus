package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	providerhandler "dpdp-backend/internal/admin/handler/emailprovider"
	userhandler "dpdp-backend/internal/admin/handler/emailuser"
	grouphandler "dpdp-backend/internal/admin/handler/group"
	policyhandler "dpdp-backend/internal/admin/handler/policy"
	rulehandler "dpdp-backend/internal/admin/handler/rule"
	providerrepo "dpdp-backend/internal/admin/repositories/emailprovider"
	userrepo "dpdp-backend/internal/admin/repositories/emailuser"
	grouprepo "dpdp-backend/internal/admin/repositories/group"
	policyrepo "dpdp-backend/internal/admin/repositories/policy"
	rulerepo "dpdp-backend/internal/admin/repositories/rule"
	providersvc "dpdp-backend/internal/admin/services/emailprovider"
	usersvc "dpdp-backend/internal/admin/services/emailuser"
	groupsvc "dpdp-backend/internal/admin/services/group"
	policysvc "dpdp-backend/internal/admin/services/policy"
	rulesvc "dpdp-backend/internal/admin/services/rule"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/middleware"
	"dpdp-backend/internal/notification"
)

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	startupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	database, err := cfg.Postgres.Connect(startupCtx, cfg.App.Env)
	if err != nil {
		slog.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}

	rdb, err := cfg.Redis.Connect(startupCtx)
	if err != nil {
		slog.Error("redis connection failed", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	notifier := notification.NewService(database, cfg.SMTP)
	store := auth.NewStore(rdb)
	service := auth.NewService(database, store, notifier, cfg.Auth, cfg.App)
	authHandler := auth.NewHandler(service, cfg.Auth)

	providerHandler := providerhandler.NewEmailProviderHandler(
		providersvc.NewEmailProviderService(
			database,
			providerrepo.NewEmailProviderRepository(database),
			providerrepo.NewRedisRepository(rdb),
			cfg.Auth,
		),
	)
	policyHandler := policyhandler.NewPolicyHandler(
		policysvc.NewPolicyService(database, policyrepo.NewPolicyRepository(database)),
	)
	ruleHandler := rulehandler.NewRuleHandler(
		rulesvc.NewRuleService(database, rulerepo.NewRuleRepository(database)),
	)
	emailUserHandler := userhandler.NewEmailUserHandler(
		usersvc.NewEmailUserService(database, userrepo.NewEmailUserRepository(database)),
	)
	groupHandler := grouphandler.NewGroupHandler(
		groupsvc.NewGroupService(database, grouprepo.NewGroupRepository(database)),
	)

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.App.AllowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")

	public := api.Group("", middleware.RequireOrigin(cfg.App))
	refresh := api.Group("", middleware.RequireOrigin(cfg.App), middleware.RequireCSRFCookie())
	protected := api.Group("",
		middleware.RequireOrigin(cfg.App),
		middleware.RequireAuth(service, store, cfg.Auth),
		middleware.RequireCSRF(cfg.Auth),
	)

	authHandler.RegisterRoutes(public, refresh, protected)

	guard := func(name string) gin.HandlerFunc {
		return middleware.RequirePrivilege(database, store, name)
	}

	providerHandler.RegisterRoutes(protected, guard)
	policyHandler.RegisterRoutes(protected, guard)
	ruleHandler.RegisterRoutes(protected, guard)
	emailUserHandler.RegisterRoutes(protected, guard)
	groupHandler.RegisterRoutes(protected, guard)

	server := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server listening", "port", cfg.App.Port, "env", cfg.App.Env)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
