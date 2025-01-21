package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"beer_oclock/internal/db"
	"beer_oclock/internal/server"
	"beer_oclock/internal/store/brewers"
	"beer_oclock/internal/store/users"

	_ "github.com/joho/godotenv/autoload" // Automatically load .env file
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

	logger.Print("Creating users store..")
	userStore := users.NewUserStore(db.New(dbPool), logger)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		logger.Fatalf("Error when hashing password: %s", err)
	}
	userStore.AddUser(context.Background(), db.AddUserParams{
		Username:     "saltytaro",
		PasswordHash: string(passwordHash),
	})

	logger.Print("Creating brewers store..")
	brewerStore := brewers.NewBrewerStore(db.New(dbPool), logger)
	brewerStore.AddBrewer(context.Background(), db.AddBrewerParams{
		Name:     "Felon's",
		Location: sql.NullString{Valid: true, String: "Brisbane"},
	})

	srv, err := server.NewServer(logger, port, userStore, brewerStore)
	if err != nil {
		logger.Fatalf("Error when creating server: %s", err)
		os.Exit(1)
	}
	if err := srv.Start(); err != nil {
		logger.Fatalf("Error when starting server: %s", err)
		os.Exit(1)
	}
}
