package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"power/internal/app"

	"github.com/zserge/lorca"
)

func main() {
	addr, err := app.ResolveAddr("127.0.0.1:18080")
	if err != nil {
		log.Fatalf("resolve addr: %v", err)
	}
	server, err := app.NewHTTPServer(addr)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("server exited: %v", err)
		}
	}()

	url := "http://" + addr
	ui, err := lorca.New(url, "", 1200, 800)
	if err != nil {
		log.Printf("桌面窗口启动失败，回退浏览器模式: %v", err)
		_ = openBrowser(url)
		waitSignalAndShutdown(server)
		return
	}
	defer ui.Close()

	<-ui.Done()
	shutdown(server)
}

func openBrowser(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func waitSignalAndShutdown(server interface{ Shutdown(context.Context) error }) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	shutdown(server)
}

func shutdown(server interface{ Shutdown(context.Context) error }) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
