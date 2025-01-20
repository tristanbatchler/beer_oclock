package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"beer_oclock/internal/db"
	"beer_oclock/internal/store"
	"beer_oclock/internal/store/contacts"
	"beer_oclock/internal/templates"
)

type server struct {
	logger     *log.Logger
	port       int
	httpServer *http.Server
	contactDb  *contacts.ContactStore
}

// Creat a new server instance with the given logger and port
func NewServer(logger *log.Logger, port int, contactDb *contacts.ContactStore) (*server, error) {
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
	router.HandleFunc("POST /contact", s.addContactHandler)
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
	contacts, err := s.contactDb.GetContacts(r.Context())
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

// POST /contact
func (s *server) addContactHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Adding contact")
	if err := r.ParseForm(); err != nil {
		s.logger.Printf("Error when parsing form: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	formData := db.Contact{Name: r.FormValue("name"), Email: r.FormValue("email")}

	validationErrors := make(map[string]string)
	if formData.Name == "" {
		validationErrors["name"] = "Name is required"
	}
	if formData.Email == "" {
		validationErrors["email"] = "Email is required"
	}
	if !strings.Contains(formData.Email, "@") {
		validationErrors["email"] = "Email is invalid"
	}
	if len(validationErrors) > 0 {
		buf := bytes.Buffer{}
		templates.AddContactForm(formData, validationErrors).Render(r.Context(), &buf)
		http.Error(w, buf.String(), http.StatusUnprocessableEntity)
		return
	}

	contact, err := s.contactDb.AddContact(r.Context(), db.AddContactParams{Name: formData.Name, Email: formData.Email})
	if err != nil {
		errMsg := fmt.Sprintf("Error when adding contact: %v", err)
		s.logger.Print(errMsg)

		validationErrors := make(map[string]string)

		switch err.(type) {
		case store.ErrMissingField:
			validationErrors[err.(store.ErrMissingField).Field] = "This field is required"
		case contacts.ErrContactAlreadyExists:
			validationErrors["email"] = "Contact with this email already exists"
		default:
			http.Error(w, errMsg, http.StatusInternalServerError)
		}
		buf := bytes.Buffer{}
		templates.AddContactForm(formData, validationErrors).Render(r.Context(), &buf)
		http.Error(w, buf.String(), http.StatusUnprocessableEntity)
		return
	}

	templates.AddContactForm(db.Contact{}, nil).Render(r.Context(), w)
	templates.ContactToAppend(contact).Render(r.Context(), w)
}

// DELETE /contact/{id}
func (s *server) deleteContactHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Deleting contact with id: %s", r.PathValue("id"))
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	_, err = s.contactDb.DeleteContact(r.Context(), int64(id))
	if err != nil {
		errMsg := fmt.Sprintf("Error when deleting contact: %v", err)
		s.logger.Print(errMsg)

		http.Error(w, errMsg, http.StatusNoContent)
		return
	}

	// Check if that was the last contact
	numContacts, err := s.contactDb.CountContacts(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when counting contacts: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	if numContacts == 0 {
		// If we just deleted the last contact, render the no contacts template
		templates.NoContacts().Render(r.Context(), w)
	} else {
		// Return nothing so the target of the delete request is replaced with nothing, i.e. removed
		w.WriteHeader(http.StatusNoContent)
	}
}
