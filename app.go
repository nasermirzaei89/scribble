package main

import (
	"context"
	"fmt"
	"net/http"
)

type App struct {
	httpServer *http.Server
}

func NewApp() *App {
	app := new(App)

	httpHandler := NewHTTPHandler()

	app.httpServer = &http.Server{
		Addr:    ":8080",
		Handler: httpHandler,
	}

	return app
}

func (app *App) Run(ctx context.Context) error {
	err := app.httpServer.ListenAndServe()
	if err != nil {
		return fmt.Errorf("failed to start http server: %w", err)
	}

	return nil
}
