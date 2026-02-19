package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/gorilla/sessions"
	_ "github.com/joho/godotenv/autoload"
	"github.com/nasermirzaei89/env"
	"github.com/nasermirzaei89/scribble/auth"
	"github.com/nasermirzaei89/scribble/contents"
	"github.com/nasermirzaei89/scribble/db/sqlite3"
	"github.com/nasermirzaei89/scribble/random"
	"github.com/nasermirzaei89/scribble/server"
)

type App struct {
	server  *server.Server
	handler *HTTPHandler
	db      *sql.DB
}

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

	authSvc := auth.NewService(userRepo, sessionRepo)
	contentsSvc := contents.NewService(postRepo)

	sessionName := env.GetString("SESSION_NAME", "scribble-"+random.String(4))
	sessionKey := env.GetString("SESSION_KEY", random.String(32))
	cookieStore := sessions.NewCookieStore([]byte(sessionKey))

	csrfAuthKeys := []byte(env.GetString("CSRF_AUTH_KEY", random.String(32)))
	csrfTrustedOrigins := env.GetStringSlice("CSRF_TRUSTED_ORIGINS", []string{})

	httpHandler, err := NewHTTPHandler(authSvc, contentsSvc, cookieStore, sessionName, csrfAuthKeys, csrfTrustedOrigins)
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
