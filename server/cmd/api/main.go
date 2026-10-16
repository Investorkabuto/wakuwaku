// Command api は wakuwaku の API サーバー。
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

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"github.com/investorkabuto/wakuwaku/server/internal/care"
	"github.com/investorkabuto/wakuwaku/server/internal/config"
	"github.com/investorkabuto/wakuwaku/server/internal/gen/wakuwaku/v1/wakuwakuv1connect"
	"github.com/investorkabuto/wakuwaku/server/internal/identity"
	"github.com/investorkabuto/wakuwaku/server/internal/platform/auth"
	"github.com/investorkabuto/wakuwaku/server/internal/platform/db"
	"github.com/investorkabuto/wakuwaku/server/internal/system"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	var verifier auth.Verifier
	if cfg.UseDevAuth() {
		slog.Warn("using development auth (dev:<uid> tokens)")
		verifier = auth.DevVerifier{}
	} else {
		verifier, err = auth.NewFirebaseVerifier(ctx, cfg.FirebaseProjectID)
		if err != nil {
			return err
		}
	}
	authInterceptor := connect.WithInterceptors(auth.NewInterceptor(verifier, identity.NewStore(pool)))

	mux := http.NewServeMux()
	mux.Handle(wakuwakuv1connect.NewSystemServiceHandler(&system.Service{Version: cfg.Version}))
	mux.Handle(wakuwakuv1connect.NewCareServiceHandler(care.NewService(pool), authInterceptor))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := cors.New(cors.Options{
		AllowedOrigins: cfg.AllowedOrigins,
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: append(connectcors.AllowedHeaders(), "Authorization"),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         7200,
	}).Handler(mux)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h2c.NewHandler(handler, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr, "env", cfg.Env)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
