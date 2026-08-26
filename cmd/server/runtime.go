package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/certification"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/credential"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/httpapi"
	"benzhi-project-d92e4a6b-b22e-4d44-8c47-1b8152b03a88/internal/ledger"
)

type application struct {
	server   *http.Server
	listener net.Listener
	service  *certification.Service
}

func assemble(address, dataDirectory string) (*application, error) {
	store, recovery, err := ledger.Open(dataDirectory, time.Now)
	if err != nil {
		return nil, fmt.Errorf("恢复账本失败: %w", err)
	}
	service, err := certification.NewService(store, recovery, credential.NewIssuer(time.Now), time.Now)
	if err != nil {
		return nil, fmt.Errorf("装配认证服务失败: %w", err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("监听 %s 失败: %w", address, err)
	}
	server := &http.Server{
		Addr: address, Handler: httpapi.New(service).Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 32 << 10,
	}
	return &application{server: server, listener: listener, service: service}, nil
}

func (a *application) serve() <-chan error {
	result := make(chan error, 1)
	go func() {
		err := a.server.Serve(a.listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		result <- err
	}()
	return result
}

func (a *application) shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

func runServer(configuration config) error {
	app, err := assemble(configuration.Address, configuration.DataDirectory)
	if err != nil {
		return err
	}
	serveResult := app.serve()
	fmt.Printf("洁净工作台认证服务监听于 http://%s\n", configuration.Address)
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serveResult:
		return err
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.shutdown(shutdownContext); err != nil {
			return fmt.Errorf("关闭 HTTP 服务: %w", err)
		}
		return <-serveResult
	}
}
