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

	alerthandler "dpdp-backend/internal/admin/handler/alert"
	brandinghandler "dpdp-backend/internal/admin/handler/branding"
	discoveryhandler "dpdp-backend/internal/admin/handler/datadiscovery"
	providerhandler "dpdp-backend/internal/admin/handler/emailprovider"
	userhandler "dpdp-backend/internal/admin/handler/emailuser"
	grouphandler "dpdp-backend/internal/admin/handler/group"
	policyhandler "dpdp-backend/internal/admin/handler/policy"
	rulehandler "dpdp-backend/internal/admin/handler/rule"
	alertrepo "dpdp-backend/internal/admin/repositories/alert"
	brandingrepo "dpdp-backend/internal/admin/repositories/branding"
	discoveryrepo "dpdp-backend/internal/admin/repositories/datadiscovery"
	providerrepo "dpdp-backend/internal/admin/repositories/emailprovider"
	userrepo "dpdp-backend/internal/admin/repositories/emailuser"
	grouprepo "dpdp-backend/internal/admin/repositories/group"
	policyrepo "dpdp-backend/internal/admin/repositories/policy"
	rulerepo "dpdp-backend/internal/admin/repositories/rule"
	alertsvc "dpdp-backend/internal/admin/services/alert"
	brandingsvc "dpdp-backend/internal/admin/services/branding"
	discoverysvc "dpdp-backend/internal/admin/services/datadiscovery"
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
	appcrypto "dpdp-backend/internal/crypto"
	discoveryprovider "dpdp-backend/internal/datadiscovery/provider"
	discoveryscanner "dpdp-backend/internal/datadiscovery/scanner"
	"dpdp-backend/internal/datadiscovery/strategy"
	"dpdp-backend/internal/delivery"
	"dpdp-backend/internal/delivery/handler/rest"
	"dpdp-backend/internal/delivery/handler/smtp"
	templaterepo "dpdp-backend/internal/delivery/repositories/emailtemplate"
	policysetrepo "dpdp-backend/internal/delivery/repositories/policyset"
	"dpdp-backend/internal/delivery/repositories/provider"
	"dpdp-backend/internal/delivery/services/adjudication"
	"dpdp-backend/internal/delivery/services/alerting"
	"dpdp-backend/internal/delivery/services/inspection"
	deliverypolicy "dpdp-backend/internal/delivery/services/policy"
	"dpdp-backend/internal/delivery/services/recording"
	"dpdp-backend/internal/delivery/services/screening"
	"dpdp-backend/internal/delivery/services/transmission"
	deliveryutils "dpdp-backend/internal/delivery/utils"
	"dpdp-backend/internal/middleware"
	"dpdp-backend/internal/notification"
	notificationworker "dpdp-backend/internal/notification/worker"
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

	if err := cfg.SMTP.Validate(); err != nil {
		slog.Error("email transport configuration invalid", "error", err)
		os.Exit(1)
	}

	if err := cfg.DataDiscovery.Validate(); err != nil {
		slog.Error("data discovery configuration invalid", "error", err)
		os.Exit(1)
	}

	secretBox, err := appcrypto.NewSecretBox(cfg.DataDiscovery.MasterKey, cfg.DataDiscovery.KeyVersion)
	if err != nil {
		slog.Error("data discovery credential encryption unavailable", "error", err)
		os.Exit(1)
	}

	notifier := notification.NewService(database, cfg.SMTP)
	store := auth.NewStore(rdb)

	notificationQueue := notificationworker.NewNotificationQueue(rdb, cfg.Notification)
	if err := notificationQueue.EnsureGroup(startupCtx); err != nil {
		slog.Error("notification consumer group creation failed", "error", err)
		os.Exit(1)
	}

	notificationWorkers := notificationworker.NewWorkerPool(
		notificationQueue,
		notificationworker.NewNotificationProcessor(notifier),
		cfg.Notification,
	)

	templateRepository := templaterepo.NewEmailTemplateRepository(database)
	alertService := alertsvc.NewAlertService(database, alertrepo.NewAlertRepository(database))

	breachAlerts := alerting.New(alerting.Options{
		Alerts:      alertLookup{service: alertService},
		Queue:       notificationQueue,
		Orgs:        templateRepository,
		FrontendURL: cfg.App.FrontendBaseURL,
		Timeout:     cfg.Notification.PublishTimeout,
	})

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
		templateRepository,
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
		Alerts:      breachAlerts,
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
	alertHandler := alerthandler.NewAlertHandler(alertService)

	providerClient := discoveryprovider.NewClient(cfg.DataDiscovery.TestTimeout)

	policyRegistry := strategy.DefaultPolicyRegistry()

	discoveryConfigurationService := discoverysvc.NewConfigurationService(
		database,
		discoveryrepo.NewConfigurationRepository(database),
		strategy.DefaultConfigurationRegistry(providerClient),
		secretBox,
		cfg.DataDiscovery.TestTimeout,
	)
	discoveryConfigurationHandler := discoveryhandler.NewConfigurationHandler(discoveryConfigurationService)
	discoveryPolicyHandler := discoveryhandler.NewPolicyHandler(
		discoverysvc.NewPolicyService(
			database,
			discoveryrepo.NewPolicyRepository(database),
			policyRegistry,
		),
	)

	discoveryScanService := discoverysvc.NewScanService(
		database,
		discoveryrepo.NewScanRepository(database),
		discoveryrepo.NewPolicyRepository(database),
		discoveryConfigurationService,
		policyRegistry,
	)
	discoveryScanHandler := discoveryhandler.NewScanHandler(discoveryScanService)

	scanner := discoveryscanner.New(discoveryscanner.Options{
		Store: discoveryScanService,
		Connect: discoveryscanner.RegistryConnector(
			policyRegistry,
			discoveryprovider.NewStreamingClient(cfg.DataDiscovery.TestTimeout),
		),
		Processors: discoveryscanner.DefaultProcessors(
			discoveryscanner.DefaultLimits(cfg.DataDiscovery.SpoolDir, cfg.DataDiscovery.MaxSpoolBytes),
		),
		QueueCapacity:    cfg.DataDiscovery.QueueCapacity,
		ChunkBytes:       cfg.DataDiscovery.ChunkBytes,
		EvaluatorWorkers: cfg.DataDiscovery.EvaluatorWorkers,
		FileTimeout:      cfg.DataDiscovery.FileTimeout,
		SpoolDir:         cfg.DataDiscovery.SpoolDir,
	})
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
	alertHandler.RegisterRoutes(protected, guard)
	discoveryConfigurationHandler.RegisterRoutes(protected, guard)
	discoveryPolicyHandler.RegisterRoutes(protected, guard)
	discoveryScanHandler.RegisterRoutes(protected, guard)
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
	notificationWorkers.Start(workerCtx)

	if cfg.DataDiscovery.ScannerEnabled {
		scanner.Start(workerCtx)
	}

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
		notificationWorkers.Wait()
		scanner.Wait()
		close(drained)
	}()

	select {
	case <-drained:
		slog.Info("in-flight deliveries and scans finished")
	case <-shutdownCtx.Done():
		slog.Warn("shutdown deadline reached with deliveries or scans in flight")
	}
}

type alertLookup struct {
	service *alertsvc.AlertService
}

func (l alertLookup) RealTimeEmailAlerts(
	ctx context.Context,
	customerID int,
	policyIDs []int,
) ([]alerting.Alert, error) {
	rows, err := l.service.RealTimeEmailAlerts(ctx, customerID, policyIDs)
	if err != nil {
		return nil, err
	}

	alerts := make([]alerting.Alert, 0, len(rows))
	for _, row := range rows {
		alerts = append(alerts, alerting.Alert{
			ID:         row.ID,
			Name:       row.Name,
			Recipients: []string(row.Target),
		})
	}

	return alerts, nil
}
