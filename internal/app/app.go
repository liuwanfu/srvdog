package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/liuwanfu/srvdog/internal/httpapi"
	webui "github.com/liuwanfu/srvdog/web"
)

func Run() error {
	cfg := DefaultConfig()
	service, err := NewService(cfg)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := service.Start(ctx); err != nil {
		return err
	}

	server := http.Server{
		Addr: cfg.ListenAddr,
		Handler: httpapi.NewServer(httpapi.Dependencies{
			Summary:      service.Summary,
			History:      service.History,
			Realtime:     service.Realtime,
			TouchViewer:  service.TouchViewer,
			SetRetention: service.SetRetention,
			Export:       service.Export,
			ClearHistory: service.ClearHistory,
			StaticFS:     webui.FS,
		}).Routes(),
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
