package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	gosmtp "github.com/emersion/go-smtp"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	brandinghandler "dpdp-backend/internal/admin/handler/branding"
	providerhandler "dpdp-backend/internal/admin/handler/emailprovider"
	userhandler "dpdp-backend/internal/admin/handler/emailuser"
	grouphandler "dpdp-backend/internal/admin/handler/group"
	policyhandler "dpdp-backend/internal/admin/handler/policy"
	rulehandler "dpdp-backend/internal/admin/handler/rule"
	brandingrepo "dpdp-backend/internal/admin/repositories/branding"
	providerrepo "dpdp-backend/internal/admin/repositories/emailprovider"
	userrepo "dpdp-backend/internal/admin/repositories/emailuser"
	grouprepo "dpdp-backend/internal/admin/repositories/group"
	policyrepo "dpdp-backend/internal/admin/repositories/policy"
	rulerepo "dpdp-backend/internal/admin/repositories/rule"
	brandingsvc "dpdp-backend/internal/admin/services/branding"
	providersvc "dpdp-backend/internal/admin/services/emailprovider"
	usersvc "dpdp-backend/internal/admin/services/emailuser"
	groupsvc "dpdp-backend/internal/admin/services/group"
	policysvc "dpdp-backend/internal/admin/services/policy"
	rulesvc "dpdp-backend/internal/admin/services/rule"
	audithandler "dpdp-backend/internal/audit/handler/deliveryaudit"
	incidenthandler "dpdp-backend/internal/audit/handler/emailincident"
	auditrepo "dpdp-backend/internal/audit/repositories/deliveryaudit"
	incidentrepo "dpdp-backend/internal/audit/repositories/emailincident"
	auditsvc "dpdp-backend/internal/audit/services/deliveryaudit"
	incidentsvc "dpdp-backend/internal/audit/services/emailincident"
	"dpdp-backend/internal/auth"
	"dpdp-backend/internal/config"
	"dpdp-backend/internal/delivery"
	"dpdp-backend/internal/delivery/handler/rest"
	"dpdp-backend/internal/delivery/handler/smtp"
	templaterepo "dpdp-backend/internal/delivery/repositories/emailtemplate"
	policysetrepo "dpdp-backend/internal/delivery/repositories/policyset"
	"dpdp-backend/internal/delivery/repositories/provider"
	"dpdp-backend/internal/delivery/services/adjudication"
	"dpdp-backend/internal/delivery/services/inspection"
	deliverypolicy "dpdp-backend/internal/delivery/services/policy"
	"dpdp-backend/internal/delivery/services/recording"
	"dpdp-backend/internal/delivery/services/screening"
	"dpdp-backend/internal/delivery/services/transmission"
	deliveryutils "dpdp-backend/internal/delivery/utils"
	"dpdp-backend/internal/middleware"
	"dpdp-backend/internal/notification"
	"dpdp-backend/internal/storage"
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

	mongoClient, err := cfg.Mongo.Connect(startupCtx)
	if err != nil {
		slog.Error("mongo connection failed", "error", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(context.Background())

	recorder := auditsvc.NewDeliveryAuditService(auditrepo.NewDeliveryAuditRepository(mongoClient, cfg.Mongo.Database))
	if err := recorder.EnsureIndexes(startupCtx); err != nil {
		slog.Error("delivery audit index creation failed", "error", err)
		os.Exit(1)
	}

	incidents := incidentsvc.NewEmailIncidentService(incidentrepo.NewEmailIncidentRepository(mongoClient, cfg.Mongo.Database))
	if err := incidents.EnsureIndexes(startupCtx); err != nil {
		slog.Error("email incident index creation failed", "error", err)
		os.Exit(1)
	}

	s3Client, err := cfg.Storage.Connect(startupCtx)
	if err != nil {
		slog.Error("object storage connection failed", "error", err)
		os.Exit(1)
	}

	objectStore := storage.New(s3Client, cfg.Storage.Bucket)

	notifier := notification.NewService(database, cfg.SMTP)
	store := auth.NewStore(rdb)

	brandingService := brandingsvc.NewBrandingService(
		brandingrepo.NewBrandingRepository(database),
		objectStore,
		store,
		cfg.Storage.LogoPresignTTL,
	)
	brandingHandler := brandinghandler.NewBrandingHandler(brandingService)

	service := auth.NewService(database, store, notifier, brandingService, cfg.Auth, cfg.App)
	authHandler := auth.NewHandler(service, cfg.Auth)

	providerRepository := providerrepo.NewEmailProviderRepository(database)
	domainRegistry := providerrepo.NewRedisRepository(rdb)

	configurations := provider.NewConfigurationStore(providerRepository)
	authorizer := smtp.NewAuthorizer(provider.NewDomainLookup(domainRegistry), configurations)

	signingConfigs := provider.NewConfigCache(configurations, rdb)
	mailRelay := transmission.NewRelay(cfg.Relay)

	blockNotice := adjudication.NewBlockNoticeService(
		templaterepo.NewEmailTemplateRepository(database),
		signingConfigs,
		mailRelay,
	)

	policyCache := deliverypolicy.NewPolicyCacheService(
		policysetrepo.NewPolicySetRepository(database),
		deliverypolicy.NewCompiler(deliveryutils.MaxRules),
		rdb,
		deliveryutils.CacheTTL,
	)

	enforcer := screening.New(screening.Options{
		Cache:       policyCache,
		Content:     inspection.NewContentEngine(inspection.DefaultMatcherFactory()),
		Actions:     adjudication.DefaultActionFactory(blockNotice),
		Incidents:   recording.NewIncidentGenerator(incidents),
		FailsClosed: deliveryutils.FailClosed,
	})

	deliveryService := delivery.NewDeliveryService(
		screening.NewEngine(signingConfigs, enforcer),
		mailRelay,
		recorder,
	)

	queue := delivery.NewQueue(rdb, cfg.Delivery)
	if err := queue.EnsureGroup(startupCtx); err != nil {
		slog.Error("delivery consumer group creation failed", "error", err)
		os.Exit(1)
	}

	workers := delivery.NewWorkerPool(queue, deliveryService, cfg.Delivery)
	acceptor := delivery.NewAcceptor(recorder, queue)

	smtpServer := smtp.NewServer(
		cfg.SMTPServer,
		smtp.NewBackend(
			authorizer,
			acceptor,
			cfg.SMTPServer.MaxSize,
			cfg.SMTPServer.MaxRecipients,
		),
		cfg.Relay.HELOHost,
	)

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
	auditHandler := audithandler.NewDeliveryAuditHandler(recorder)
	incidentHandler := incidenthandler.NewEmailIncidentHandler(incidents)

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
	brandingHandler.RegisterRoutes(protected, guard)
	auditHandler.RegisterRoutes(protected, guard)
	incidentHandler.RegisterRoutes(protected, guard)

	rest.NewSubmitHandler(
		authorizer,
		acceptor,
		cfg.SMTPServer.MaxSize,
		cfg.SMTPServer.MaxRecipients,
	).RegisterRoutes(api)

	slog.Warn("unauthenticated mail submission route enabled",
		"route", "POST /api/v1/delivery/messages",
		"reason", "port 25 stopgap",
	)

	server := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()

	workers.Start(workerCtx)

	var servers sync.WaitGroup

	servers.Add(2)

	go func() {
		defer servers.Done()

		slog.Info("api listening", "port", cfg.App.Port, "env", cfg.App.Env)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("api server failed", "error", err)
			stop()
		}
	}()

	go func() {
		defer servers.Done()

		slog.Info("smtp listening", "addr", smtpServer.Addr())

		if err := smtpServer.ListenAndServe(); err != nil && !errors.Is(err, gosmtp.ErrServerClosed) {
			slog.Error("smtp server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.SMTPServer.ShutdownTimeout)
	defer shutdownCancel()

	if err := smtpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("smtp shutdown failed", "error", err)
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("api shutdown failed", "error", err)
	}

	servers.Wait()

	stopWorkers()

	drained := make(chan struct{})

	go func() {
		workers.Wait()
		close(drained)
	}()

	select {
	case <-drained:
		slog.Info("in-flight deliveries finished")
	case <-shutdownCtx.Done():
		slog.Warn("shutdown deadline reached with deliveries in flight")
	}
}
