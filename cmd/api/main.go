package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/xXMateus97Xx/email-pixel-tracker/api"
	"github.com/xXMateus97Xx/email-pixel-tracker/recaptcha"
	"github.com/xXMateus97Xx/email-pixel-tracker/repository/factory"
	"github.com/xXMateus97Xx/email-pixel-tracker/service"
	"github.com/xXMateus97Xx/email-pixel-tracker/telemetry"
)

const defaultTokenTTL = time.Hour

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	healthcheck := flag.Bool("healthcheck", false,
		"check /health/ready of the running API and exit with 0 when ready (for container health checks)")
	flag.Parse()

	// Variables already set in the environment take precedence over the file.
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "loading .env:", err)
		os.Exit(1)
	}

	if *healthcheck {
		if err := checkHealth(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	logger, err := telemetry.NewLogger(os.Stdout, os.Getenv("LOG_LEVEL"), os.Getenv("LOG_FORMAT"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Also routes the standard log package (used by libraries) to the logger.
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	bcryptCost := 0 // service default
	if raw := os.Getenv("BCRYPT_COST"); raw != "" {
		cost, err := strconv.Atoi(raw)
		if err != nil {
			return errors.New("BCRYPT_COST must be an integer")
		}
		bcryptCost = cost
	}

	tokenTTL := defaultTokenTTL
	if raw := os.Getenv("JWT_TTL"); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil {
			return errors.New("JWT_TTL must be a duration, e.g. 1h or 30m")
		}
		tokenTTL = ttl
	}
	tokens, err := api.NewTokenManager(os.Getenv("JWT_SECRET"), tokenTTL)
	if err != nil {
		return err
	}

	// Docs are off unless explicitly enabled.
	docsEnabled := false
	if raw := os.Getenv("DOCS_ENABLED"); raw != "" {
		if docsEnabled, err = strconv.ParseBool(raw); err != nil {
			return errors.New("DOCS_ENABLED must be true or false")
		}
	}

	verifier, err := recaptchaFromEnv()
	if err != nil {
		return err
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tracingEnabled, shutdownTracing, err := telemetry.SetupTracing(startupCtx, version)
	if err != nil {
		return fmt.Errorf("configuring tracing: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(ctx); err != nil {
			logger.Warn("flushing traces", "error", err)
		}
	}()

	metrics := telemetry.NewMetrics()
	metricsAddr := os.Getenv("METRICS_ADDR")
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	dbDriver := os.Getenv("DB_DRIVER")
	logger.Info("starting",
		"version", version,
		"db_driver", dbDriver,
		"http_addr", addr,
		"metrics_addr", metricsAddr,
		"tracing_enabled", tracingEnabled,
		"docs_enabled", docsEnabled,
		"recaptcha_enabled", verifier.Enabled(),
		"bcrypt_cost", bcryptCost,
	)

	repos, err := factory.New(startupCtx, factory.Config{
		Driver:   dbDriver,
		DSN:      os.Getenv("DB_DSN"),
		Database: os.Getenv("DB_NAME"),
		Logger:   logger,
		Metrics:  metrics,
	})
	if err != nil {
		return err
	}
	defer repos.Close(context.Background())

	services, err := service.New(repos, service.Config{BcryptCost: bcryptCost})
	if err != nil {
		return err
	}

	// Without a SuperAdmin nobody could create the first tenant.
	if username := os.Getenv("SUPERADMIN_USERNAME"); username != "" {
		name := os.Getenv("SUPERADMIN_NAME")
		if name == "" {
			name = "Super Admin"
		}
		created, err := services.Auth().EnsureSuperAdmin(startupCtx, username, name, os.Getenv("SUPERADMIN_PASSWORD"))
		if err != nil {
			return fmt.Errorf("creating the first SuperAdmin: %w", err)
		}
		if created {
			logger.Info("SuperAdmin created", "username", username)
		}
	}

	var trustedProxies []string
	for _, proxy := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			trustedProxies = append(trustedProxies, proxy)
		}
	}

	// Requests are logged by the API itself; gin's debug output is only noise.
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	router, err := api.NewRouter(services, tokens, api.RouterConfig{
		TrustedProxies: trustedProxies,
		DocsEnabled:    docsEnabled,
		DocsPath:       os.Getenv("DOCS_PATH"),
		Metrics:        metrics,
		Readiness:      repos.Ping,
		Recaptcha:      verifier,
	})
	if err != nil {
		return fmt.Errorf("configuring routes: %w", err)
	}

	servers := []*http.Server{{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}}
	// Metrics have their own address, kept off the public port.
	if metricsAddr != "" {
		mux := http.NewServeMux()
		mux.Handle("GET /metrics", metrics.Handler())
		servers = append(servers, &http.Server{
			Addr:              metricsAddr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		})
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, len(servers))
	for _, server := range servers {
		go func() {
			logger.Info("listening", "addr", server.Addr)
			serverErr <- server.ListenAndServe()
		}()
	}

	var runErr error
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}
	case <-ctx.Done():
		logger.Info("shutting down")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	for _, server := range servers {
		if err := server.Shutdown(shutdownCtx); err != nil && runErr == nil {
			runErr = err
		}
	}
	return runErr
}

// recaptchaFromEnv reads RECAPTCHA_*; reCAPTCHA is off unless RECAPTCHA_ENABLED=true.
func recaptchaFromEnv() (*recaptcha.Verifier, error) {
	enabled := false
	if raw := os.Getenv("RECAPTCHA_ENABLED"); raw != "" {
		var err error
		if enabled, err = strconv.ParseBool(raw); err != nil {
			return nil, errors.New("RECAPTCHA_ENABLED must be true or false")
		}
	}
	minScore := 0.0 // package default
	if raw := os.Getenv("RECAPTCHA_MIN_SCORE"); raw != "" {
		score, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, errors.New("RECAPTCHA_MIN_SCORE must be a number between 0 and 1, e.g. 0.5")
		}
		minScore = score
	}
	return recaptcha.New(recaptcha.Config{
		Enabled:   enabled,
		SiteKey:   os.Getenv("RECAPTCHA_SITE_KEY"),
		SecretKey: os.Getenv("RECAPTCHA_SECRET_KEY"),
		MinScore:  minScore,
	})
}

// checkHealth asks the API running on HTTP_ADDR whether it is ready. The
// container image has no curl or wget, so the binary checks itself.
func checkHealth() error {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR: %w", err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get("http://" + net.JoinHostPort(host, port) + "/health/ready")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("not ready: status %d", res.StatusCode)
	}
	return nil
}
