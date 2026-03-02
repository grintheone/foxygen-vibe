package main

import (
	"log"
	"net/http"

	"foxygen-vibe/server/internal/api"
)

func main() {
	app, err := api.New()
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	addr := ":8080"
	log.Printf("server listening on %s", addr)

	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
