package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"beer_oclock/internal/store"
	"beer_oclock/internal/templates"
)

type ContactStore interface {
	AddContact(Contact store.Contact) error
	GetContacts() ([]store.Contact, error)
	DeleteContact(id int) error
}

type server struct {
	logger     *log.Logger
	port       int
	httpServer *http.Server
	contactDb  ContactStore
}

// Creat a new server instance with the given logger and port
func NewServer(logger *log.Logger, port int, contactDb ContactStore) (*server, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if contactDb == nil {
		return nil, fmt.Errorf("contactDb is required")
	}
	return &server{
		logger:    logger,
		port:      port,
		contactDb: contactDb}, nil
}

// Start the server
func (s *server) Start() error {
	s.logger.Printf("Starting server on port %d", s.port)
	var stopChan chan os.Signal

	// define router
	router := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./static"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	router.HandleFunc("GET /", s.defaultHandler)
	router.HandleFunc("DELETE /contact/{id}", s.deleteContactHandler)

	// define server
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: router}

	// create channel to listen for signals
	stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error when running server: %s", err)
		}
	}()

	<-stopChan

	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		log.Fatalf("Error when shutting down server: %v", err)
		return err
	}
	return nil
}

// GET /
func (s *server) defaultHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	contacts, err := s.contactDb.GetContacts()
	if err != nil {
		s.logger.Printf("Error when getting contacts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	homeTemplate := templates.Home(contacts)
	err = templates.Layout(homeTemplate, "Beer O'clock", "/").Render(r.Context(), w)
	if err != nil {
		s.logger.Printf("Error when rendering home: %v", err)
	}
}

// DELETE /contact/{id}
func (s *server) deleteContactHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Deleting contact with id: %s", r.PathValue("id"))
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		s.logger.Printf("Error when converting id to int: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	err = s.contactDb.DeleteContact(id)
	if err != nil {
		s.logger.Printf("Error when deleting contact: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return nothing so the target of the delete request is replaced with nothing, i.e. removed
}
