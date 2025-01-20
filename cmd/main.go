package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"

	"beer_oclock/internal/db"
	"beer_oclock/internal/server"
	"beer_oclock/internal/store/contacts"
)

func main() {
	logger := log.New(os.Stdout, "[Main] ", log.LstdFlags)

	port := 9000

	dbPool, err := sql.Open("sqlite", "db.sqlite")
	if err != nil {
		logger.Fatalf("Error when opening database: %s", err)
	}

	log.Println("Initializing database...")
	if err := db.GenSchema(dbPool); err != nil {
		log.Fatal(err)
	}

	logger.Print("Creating guests store..")
	guestDb := contacts.NewContactStore(db.New(dbPool), logger)
	guestDb.AddContact(context.Background(), db.AddContactParams{
		Name:  "John Doe",
		Email: "jdoe@mail.net",
	})

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
