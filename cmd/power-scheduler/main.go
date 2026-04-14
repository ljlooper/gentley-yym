package main

import (
	"log"

	"power/internal/app"
)

func main() {
	addr, err := app.ResolveAddr("127.0.0.1:8080")
	if err != nil {
		log.Fatalf("resolve addr: %v", err)
	}
	server, err := app.NewHTTPServer(addr)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}
	log.Printf("Power Scheduler running at http://%s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
