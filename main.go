package main

import (
	"context"
	"fmt"
	authDB "game-night/auth/db"
	authSqlc "game-night/auth/db/sqlc"
	authService "game-night/auth/services"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	db, err := NewDBConnPool(ctx, config.DBSource)
	if err != nil {
		return err
	}

	// instantiate related queries and store
	authQueries := authSqlc.New(db)
	authStore := authDB.NewAuthStore(authQueries)
	authService := authService.NewAuthService(authStore)

	logger := NewAppLogger()
	srv := NewServer(logger, authService)

	httpServer := http.Server{
		Addr:    net.JoinHostPort(config.Host, config.Port),
		Handler: srv,
	}

	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error listening and serving :%s\n", err)
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()

	wg.Wait()
	return nil
}
