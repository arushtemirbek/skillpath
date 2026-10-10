package main

import (
	"log"

	"github.com/arushtemirbek/skillpath/internal/server"
)

func main() {
	srv := server.New()

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
