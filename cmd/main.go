package main

import (
	"log"
	"os"

	"beer_oclock/internal/server"
	"beer_oclock/internal/store"
)

func main() {
	logger := log.New(os.Stdout, "[Main] ", log.LstdFlags)

	port := 9000

	logger.Print("Creating guests store..")
	guestDb := store.NewContactStore(logger)
	guestDb.AddContact(store.Contact{Name: "John", Email: "john@mail.net"})

	srv, err := server.NewServer(logger, port, guestDb)
	if err != nil {
		logger.Fatalf("Error when creating server: %s", err)
		os.Exit(1)
	}
	if err := srv.Start(); err != nil {
		logger.Fatalf("Error when starting server: %s", err)
		os.Exit(1)
	}
}
