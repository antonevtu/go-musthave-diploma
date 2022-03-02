package app

import (
	"context"
	"github.com/antonevtu/go-musthave-diploma/internal/accrual"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
	"github.com/antonevtu/go-musthave-diploma/internal/handlers"
	"github.com/antonevtu/go-musthave-diploma/internal/logger"
	"github.com/antonevtu/go-musthave-diploma/internal/repository"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run() {
	zLog, err := logger.New(0)
	if err != nil {
		log.Fatal(err)
	}
	zLog.Infow("starting service...")

	cfgApp, err := cfg.New()
	if err != nil {
		zLog.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = context.WithValue(ctx, logger.Z, zLog)

	// database
	dbPool, err := repository.NewDB(ctx, cfgApp.DatabaseURI)
	if err != nil {
		zLog.Fatal(err)
	}
	defer dbPool.Close()
	repo := &dbPool

	// accrual pool
	accrualPool := accrual.New(ctx, repo, cfgApp)
	defer accrualPool.Close()

	r := handlers.NewRouter(repo, cfgApp)
	httpServer := &http.Server{
		Addr:        cfgApp.RunAddress,
		Handler:     r,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	// Run server
	go func() {
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			zLog.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
	)

	select {
	case <-signalChan:
		zLog.Infow("os.Interrupt - shutting down...")
	case err := <-accrualPool.ErrCh:
		zLog.Infow(err.Error())
	}
	cancel()

	gracefulCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err = httpServer.Shutdown(gracefulCtx); err != nil {
		zLog.Infow("shutdown error: %v\n", err)
	} else {
		zLog.Infow("web server gracefully stopped")
	}
}
