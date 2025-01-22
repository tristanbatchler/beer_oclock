package server

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"beer_oclock/internal/db"
	"beer_oclock/internal/middleware"
	"beer_oclock/internal/store"
	"beer_oclock/internal/store/brewers"
	"beer_oclock/internal/store/users"
	"beer_oclock/internal/templates"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

type server struct {
	logger       *log.Logger
	port         int
	httpServer   *http.Server
	userStore    *users.UserStore
	brewerStore  *brewers.BrewerStore
	sessionStore *BeerOclockSessionStore
}

// Creat a new server instance with the given logger and port
func NewServer(logger *log.Logger, port int, userStore *users.UserStore, brewerStore *brewers.BrewerStore) (*server, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if userStore == nil {
		return nil, fmt.Errorf("userStore is required")
	}
	if brewerStore == nil {
		return nil, fmt.Errorf("brewerStore is required")
	}

	sessionKeyB64 := os.Getenv("SESSION_KEY")
	if sessionKeyB64 == "" {
		return nil, fmt.Errorf("SESSION_KEY is required as a base64 encoded string of 32 random bytes")
	}

	sessionKeyBytes, err := base64.StdEncoding.DecodeString(sessionKeyB64)
	if err != nil {
		return nil, fmt.Errorf("Error when decoding session key. Ensure it is a base64 encoded string of 32 random bytes: %v", err)
	}

	cookieStore := sessions.NewCookieStore(sessionKeyBytes)

	return &server{
		logger:       logger,
		port:         port,
		userStore:    userStore,
		brewerStore:  brewerStore,
		sessionStore: NewBeerOclockSessionStore(cookieStore, userStore),
	}, nil
}

// Start the server
func (s *server) Start() error {
	s.logger.Printf("Starting server on port %d", s.port)
	var stopChan chan os.Signal

	// define router
	router := http.NewServeMux()

	// define middleware
	authMiddleware := middleware.Auth(s.sessionStore, s.userStore)
	loggingMiddleware := middleware.Chain(middleware.ContentType, middleware.Logging)
	authLoggingMiddleware := middleware.Chain(middleware.ContentType, middleware.Logging, authMiddleware)

	// unprotected routes:
	fileServer := http.FileServer(http.Dir("./static"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	router.Handle("GET /favicon.ico", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/images/favicon/favicon.ico")
	}))

	router.Handle("GET /login", loggingMiddleware(http.HandlerFunc(s.loginFormHandler)))
	router.Handle("POST /login", loggingMiddleware(http.HandlerFunc(s.loginHandler)))

	// protected routes:
	router.Handle("GET /", authLoggingMiddleware(http.HandlerFunc(s.homeHandler)))

	router.Handle("GET /logout", authLoggingMiddleware(http.HandlerFunc(s.logoutHandler)))
	router.Handle("POST /logout", authLoggingMiddleware(http.HandlerFunc(s.logoutHandler)))

	router.Handle("POST /brewer", authLoggingMiddleware(http.HandlerFunc(s.addBrewerHandler)))
	router.Handle("GET /brewer/add", authLoggingMiddleware(http.HandlerFunc(s.getBrewerFormHandler)))
	router.Handle("DELETE /brewer/{id}", authLoggingMiddleware(http.HandlerFunc(s.deleteBrewerHandler)))
	router.Handle("GET /brewers", authLoggingMiddleware(http.HandlerFunc(s.listBrewersHandler)))
	router.Handle("GET /brewer/{id}", authLoggingMiddleware(http.HandlerFunc(s.getBrewerHandler)))

	router.Handle("POST /user", authLoggingMiddleware(http.HandlerFunc(s.addUserHandler)))
	router.Handle("GET /user/add", authLoggingMiddleware(http.HandlerFunc(s.getUserFormHandler)))
	router.Handle("DELETE /user/{id}", authLoggingMiddleware(http.HandlerFunc(s.deleteUserHandler)))
	router.Handle("GET /users", authLoggingMiddleware(http.HandlerFunc(s.listUsersHandler)))
	router.Handle("GET /user/{id}", authLoggingMiddleware(http.HandlerFunc(s.getUserHandler)))

	// define server
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: router,
	}

	// create channel to listen for signals
	stopChan = make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error when running server: %s", err)
		}
	}()

	<-stopChan

	// Create a context with a timeout of 5 seconds
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Error when shutting down server: %v", err)
		return err
	}
	return nil
}

// GET /
func (s *server) homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	homeTemplate := templates.Home()
	err := templates.Layout(homeTemplate, "Beer O'Clock").Render(r.Context(), w)
	if err != nil {
		s.logger.Printf("Error when rendering home: %v", err)
	}
}

// GET /login
func (s *server) loginFormHandler(w http.ResponseWriter, r *http.Request) {
	// Pass through if already logged in
	if _, err := s.sessionStore.ValidateSession(r); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	loginTemplate := templates.LoginForm(nil)
	err := templates.Layout(loginTemplate, "Beer O'Clock").Render(r.Context(), w)
	if err != nil {
		s.logger.Printf("Error when rendering login form: %v", err)
	}
}

// GET or POST /logout
func (s *server) logoutHandler(w http.ResponseWriter, r *http.Request) {

	s.sessionStore.EraseCurrent(w, r)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// POST /brewer
func (s *server) addBrewerHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Adding brewer")
	if err := r.ParseForm(); err != nil {
		s.logger.Printf("Error when parsing form: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	formLocation := r.FormValue("location")
	locationNullString := sql.NullString{}
	if formLocation != "" {
		locationNullString = sql.NullString{Valid: true, String: formLocation}
	}
	formData := db.Brewer{Name: r.FormValue("name"), Location: locationNullString}

	validationErrors := make(map[string]string)
	if formData.Name == "" {
		validationErrors["name"] = "Name is required"
	}
	if !formData.Location.Valid {
		validationErrors["location"] = "Location is required"
	}
	if len(validationErrors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		templates.AddBrewerForm(formData, validationErrors).Render(r.Context(), w)
		return
	}

	brewer, err := s.brewerStore.AddBrewer(r.Context(), db.AddBrewerParams{Name: formData.Name, Location: formData.Location})
	if err != nil {
		errMsg := fmt.Sprintf("Error when adding brewer: %v", err)
		s.logger.Print(errMsg)

		validationErrors := make(map[string]string)

		switch err := err.(type) {
		case store.ErrMissingField:
			validationErrors[err.Field] = "This field is required"
			w.WriteHeader(http.StatusUnprocessableEntity)
		case brewers.ErrBrewerAlreadyExists:
			validationErrors["name"] = fmt.Sprintf("%s already exists", err.Name)
			w.WriteHeader(http.StatusConflict)
		default:
			http.Error(w, errMsg, http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
		}
		templates.AddBrewerForm(formData, validationErrors).Render(r.Context(), w)
		return
	}

	templates.AddBrewerForm(db.Brewer{}, nil).Render(r.Context(), w)
	templates.BrewerToAppend(brewer).Render(r.Context(), w)
}

// GET /brewer/add
func (s *server) getBrewerFormHandler(w http.ResponseWriter, r *http.Request) {
	templates.AddBrewerForm(db.Brewer{}, nil).Render(r.Context(), w)
}

// DELETE /brewer/{id}
func (s *server) deleteBrewerHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Deleting brewer with id: %s", r.PathValue("id"))
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	_, err = s.brewerStore.DeleteBrewer(r.Context(), int64(id))
	if err != nil {
		errMsg := fmt.Sprintf("Error when deleting brewer: %v", err)
		s.logger.Print(errMsg)

		http.Error(w, errMsg, http.StatusNoContent)
		return
	}

	// Check if that was the last brewer
	numBrewers, err := s.brewerStore.CountBrewers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when counting brewers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	if numBrewers == 0 {
		// If we just deleted the last brewer, render the no brewers template
		templates.NoBrewers().Render(r.Context(), w)
	} else {
		// Return nothing so the target of the delete request is replaced with nothing, i.e. removed
		w.WriteHeader(http.StatusNoContent)
	}
}

// GET /brewers
func (s *server) listBrewersHandler(w http.ResponseWriter, r *http.Request) {
	brewers, err := s.brewerStore.GetBrewers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting brewers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	templates.BrewersList(brewers).Render(r.Context(), w)
}

// GET /brewer/{id}
func (s *server) getBrewerHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	brewer, err := s.brewerStore.GetBrewer(r.Context(), int64(id))
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting brewer: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	templates.Brewer(brewer).Render(r.Context(), w)
}

// POST /user
func (s *server) addUserHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Adding user")
	if err := r.ParseForm(); err != nil {
		s.logger.Printf("Error when parsing form: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	formUsername := r.FormValue("username")
	formPassword := r.FormValue("password")
	formConfirmPassword := r.FormValue("confirm-password")

	validationErrors := make(map[string]string)
	if formUsername == "" {
		validationErrors["username"] = "Username is required"
	}
	if formPassword == "" {
		validationErrors["password"] = "Password is required"
	}
	if formConfirmPassword == "" {
		validationErrors["confirm-password"] = "Confirm password is required"
	}
	if formPassword != formConfirmPassword {
		validationErrors["confirm-password"] = "Passwords do not match"
	}
	if len(validationErrors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		templates.AddUserForm(db.User{Username: formUsername}, validationErrors).Render(r.Context(), w)
		return
	}

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(formPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Fatalf("Error when hashing password: %s", err)
	}

	// Add the user to the user store
	s.userStore.AddUser(context.Background(), db.AddUserParams{
		Username:     formUsername,
		PasswordHash: string(passwordHash),
	})

	templates.AddUserForm(db.User{}, nil).Render(r.Context(), w)
}

// GET /user/add
func (s *server) getUserFormHandler(w http.ResponseWriter, r *http.Request) {
	templates.AddUserForm(db.User{}, nil).Render(r.Context(), w)
}

// DELETE /user/{id}
func (s *server) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Deleting user with id: %s", r.PathValue("id"))
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	_, err = s.userStore.DeleteUser(r.Context(), int64(id))
	if err != nil {
		errMsg := fmt.Sprintf("Error when deleting user: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	// Check if that was the last user
	numUsers, err := s.userStore.CountUsers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when counting users: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	if numUsers == 0 {
		// If we just deleted the last user, render the no users template
		templates.NoUsers().Render(r.Context(), w)
	} else {
		// Return nothing so the target of the delete request is replaced with nothing, i.e. removed
		w.WriteHeader(http.StatusNoContent)
	}
}

// GET /users
func (s *server) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := s.userStore.GetUsers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting users: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	templates.UsersList(users).Render(r.Context(), w)
}

// GET /user/{id}
func (s *server) getUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	user, err := s.userStore.GetUser(r.Context(), int64(id))
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting user: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	templates.User(user).Render(r.Context(), w)
}

// POST /login
func (s *server) loginHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Logging in")
	if err := r.ParseForm(); err != nil {
		s.logger.Printf("Error when parsing form: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	formUsername := r.FormValue("username")
	formPassword := r.FormValue("password")

	validationErrors := make(map[string]string)
	if formUsername == "" {
		validationErrors["username"] = "Username is required"
	}
	if formPassword == "" {
		validationErrors["password"] = "Password is required"
	}
	if len(validationErrors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		templates.LoginForm(validationErrors).Render(r.Context(), w)
		return
	}

	// Check if the user exists
	user, err := s.userStore.GetUserByUsername(r.Context(), strings.ToLower(formUsername))
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting user by username: %v", err)
		switch err.(type) {
		case users.ErrUserNotFound:
			validationErrors["password"] = "Username or password is incorrect"
			w.WriteHeader(http.StatusUnauthorized)
		default:
			validationErrors["password"] = "Internal server error"
			w.WriteHeader(http.StatusInternalServerError)
		}
		s.logger.Print(errMsg)
		templates.LoginForm(validationErrors).Render(r.Context(), w)
		return
	}

	// Check if the password is correct
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(formPassword))
	if err != nil {
		validationErrors["password"] = "Username or password is incorrect"
		w.WriteHeader(http.StatusUnauthorized)
		templates.LoginForm(validationErrors).Render(r.Context(), w)
		return
	}

	// Generate a session token
	err = s.sessionStore.WriteNew(w, r, user.ID)

	if err != nil {
		errMsg := fmt.Sprintf("Error when saving session: %v", err)
		s.logger.Print(errMsg)
		w.WriteHeader(http.StatusInternalServerError)
		validationErrors["password"] = "Internal server error"
		templates.LoginForm(validationErrors).Render(r.Context(), w)
		return
	}

	s.userStore.SetUserLastLogin(r.Context(), user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
