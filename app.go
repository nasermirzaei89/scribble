package scribble

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/gorilla/sessions"
	"github.com/nasermirzaei89/env"
	"github.com/nasermirzaei89/scribble/authentication"
	"github.com/nasermirzaei89/scribble/authorization"
	"github.com/nasermirzaei89/scribble/authorization/casbin"
	"github.com/nasermirzaei89/scribble/contents"
	"github.com/nasermirzaei89/scribble/database/sqlite3"
	"github.com/nasermirzaei89/scribble/discuss"
	"github.com/nasermirzaei89/scribble/random"
	"github.com/nasermirzaei89/scribble/reactions"
	"github.com/nasermirzaei89/scribble/server"
	"github.com/nasermirzaei89/scribble/web"
)

type App struct {
	server  *server.Server
	handler *web.Handler
	db      *sql.DB
}

//go:embed policy.csv
var defaultAuthorizationPolicyContent string

func NewApp(ctx context.Context) (*App, error) {
	db, err := sqlite3.NewDB(ctx, env.GetString("DB_DSN", "file::memory:?cache=shared"))
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	err = sqlite3.MigrateUp(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	userRepo := sqlite3.NewUserRepository(db)
	sessionRepo := sqlite3.NewSessionRepository(db)
	postRepo := sqlite3.NewPostRepository(db)
	commentRepo := sqlite3.NewCommentRepository(db)
	userReactionRepo := sqlite3.NewUserReactionRepository(db)

	authzProvider, err := newAuthorizationProvider(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization provider: %w", err)
	}

	authzSvc, err := authorization.NewService(authzProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization service: %w", err)
	}

	authzClient := authorization.NewClient(authzSvc)
	authSvc := authentication.NewService(userRepo, sessionRepo, authzClient)

	if err := authSvc.LoadBloomFilter(ctx, 10_000, 0.01); err != nil {
		return nil, fmt.Errorf("failed to load bloom filter: %w", err)
	}

	contentsSvc := contents.NewService(postRepo, authzClient)
	discussSvc := discuss.NewService(commentRepo, authzClient)
	reactionsSvc := reactions.NewService(userReactionRepo, authzClient)

	sessionName := env.GetString("SESSION_NAME", "scribble-"+random.String(4))
	sessionKey := env.GetString("SESSION_KEY", random.String(32))
	cookieStore := sessions.NewCookieStore([]byte(sessionKey))

	csrfAuthKeys := []byte(env.GetString("CSRF_AUTH_KEY", random.String(32)))
	csrfTrustedOrigins := env.GetStringSlice("CSRF_TRUSTED_ORIGINS", []string{})

	httpHandler, err := web.NewHandler(
		authSvc,
		contentsSvc,
		discussSvc,
		reactionsSvc,
		cookieStore,
		sessionName,
		csrfAuthKeys,
		csrfTrustedOrigins,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP handler: %w", err)
	}

	app := &App{
		server:  newServer(),
		handler: httpHandler,
		db:      db,
	}

	return app, nil
}

func (app *App) Run(ctx context.Context) error {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	defer func() {
		if app.db != nil {
			err := app.db.Close()
			if err != nil {
				slog.ErrorContext(ctx, "failed to close database", "error", err)
			}
		}
	}()

	err := app.server.Run(ctx, app.handler)
	if err != nil {
		return fmt.Errorf("failed to run server: %w", err)
	}

	return nil
}

func newServer() *server.Server {
	server := &server.Server{
		Port: env.GetString("PORT", server.DefaultPort),
		Host: env.GetString("HOST", ""),
		TLS: server.ServerTLS{
			Enabled: env.GetBool("TLS_ENABLED", false),
			Mode:    env.GetString("TLS_MODE", server.DefaultTLSMode),
			AutoCert: &server.ServerTLSAutoCert{
				CacheDir: env.GetString("TLS_AUTOCERT_CACHE_DIR", "./cert-cache"),
				Domains:  env.GetStringSlice("TLS_AUTOCERT_DOMAINS", []string{}),
				Email:    env.GetString("TLS_AUTOCERT_EMAIL", ""),
			},
			CertFile: env.GetString("TLS_CERT_FILE", ""),
			KeyFile:  env.GetString("TLS_KEY_FILE", ""),
		},
	}

	return server
}

func GetLogLevelFromEnv() slog.Level {
	levelStr := env.GetString("LOG_LEVEL", "info")
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		slog.Warn("unknown log level, defaulting to info", "level", levelStr)

		return slog.LevelInfo
	}
}

func newAuthorizationProvider(ctx context.Context, db *sql.DB) (*casbin.AuthorizationProvider, error) {
	adapter, err := casbin.NewSQLAdapter(db, "sqlite3", "casbin_rule")
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization adapter: %w", err)
	}

	provider, err := casbin.NewAuthorizationProvider(adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization provider: %w", err)
	}

	policyContent, err := loadPolicyContent()
	if err != nil {
		return nil, fmt.Errorf("failed to load authorization policy content: %w", err)
	}

	err = provider.AddPolicyFromCSV(ctx, policyContent)
	if err != nil {
		return nil, fmt.Errorf("failed to add authorization policy from csv: %w", err)
	}

	return provider, nil
}

func loadPolicyContent() (string, error) {
	policyFilePath := env.GetString("AUTHORIZATION_POLICY_FILE", "")

	if policyFilePath == "" {
		return defaultAuthorizationPolicyContent, nil
	}

	content, err := os.ReadFile(policyFilePath) // nolint:gosec
	if err != nil {
		return "", fmt.Errorf("failed to read policy file %q: %w", policyFilePath, err)
	}

	return string(content), nil
}
