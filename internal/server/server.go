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
	"beer_oclock/internal/store/beers"
	"beer_oclock/internal/store/brewers"
	"beer_oclock/internal/store/users"
	"beer_oclock/internal/templates"

	"github.com/a-h/templ"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

const AppName = "Beer O'clock"

type server struct {
	logger       *log.Logger
	port         int
	httpServer   *http.Server
	userStore    *users.UserStore
	brewerStore  *brewers.BrewerStore
	beerStore    *beers.BeerStore
	sessionStore *BeerOclockSessionStore
}

// Creat a new server instance with the given logger and port
func NewServer(logger *log.Logger, port int, userStore *users.UserStore, brewerStore *brewers.BrewerStore, beerStore *beers.BeerStore) (*server, error) {
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
		beerStore:    beerStore,
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

	router.Handle("POST /beer", authLoggingMiddleware(http.HandlerFunc(s.addBeerHandler)))
	router.Handle("GET /beer/add", authLoggingMiddleware(http.HandlerFunc(s.getBeerFormHandler)))
	router.Handle("DELETE /beer/{id}", authLoggingMiddleware(http.HandlerFunc(s.deleteBeerHandler)))
	router.Handle("GET /beers", authLoggingMiddleware(http.HandlerFunc(s.listBeersHandler)))
	router.Handle("GET /beer/{id}", authLoggingMiddleware(http.HandlerFunc(s.getBeerHandler)))
	router.Handle("GET /beer/{id}/edit", authLoggingMiddleware(http.HandlerFunc(s.getBeerFormHandler)))
	router.Handle("PUT /beer/{id}", authLoggingMiddleware(http.HandlerFunc(s.updateBeerHandler)))
	router.Handle("POST /beer/search", authLoggingMiddleware(http.HandlerFunc(s.searchBeersHandler)))

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

// A helper function to determine whether a request was made by HTMX, so we can use this to inform
// whether the response should be a full layout page or just the partial content
func isHtmxRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// A helper function to respond with a template, either as a full page or just the partial content
// depending on whether the request was made by HTMX and the HTML verb used (full pages only apply
// to GET requests) the AppName to the title provided. If the template fails to render, a 500 error
// is returned.
func renderTemplate(w http.ResponseWriter, r *http.Request, t templ.Component, title ...string) {
	// Return a partial response if the request was made by HTMX or if the request was not a GET request
	if isHtmxRequest(r) || r.Method != http.MethodGet {
		t.Render(r.Context(), w)
		return
	}

	// Otherwise, format the title
	if len(title) <= 0 {
		title = append(title, AppName)
	} else {
		title[0] = fmt.Sprintf("%s ~ %s", title[0], AppName)
	}

	// and render the full page
	err := templates.Layout(t, title[0]).Render(r.Context(), w)
	if err != nil {
		log.Printf("Error when rendering: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GET /
func (s *server) homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	beers, err := s.beerStore.GetBeers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting beers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	renderTemplate(w, r, templates.Home(beers), "Home")
}

// GET /login
func (s *server) loginFormHandler(w http.ResponseWriter, r *http.Request) {
	// Pass through if already logged in
	if _, err := s.sessionStore.ValidateSession(r); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	renderTemplate(w, r, templates.LoginForm(nil), "Login")
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
		renderTemplate(w, r, templates.AddBrewerForm(formData, validationErrors))
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
		renderTemplate(w, r, templates.AddBrewerForm(formData, validationErrors))
		return
	}

	renderTemplate(w, r, templates.AddBrewerForm(db.Brewer{}, nil))
	renderTemplate(w, r, templates.Brewer(brewer))
}

// GET /brewer/add
func (s *server) getBrewerFormHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, templates.AddBrewerForm(db.Brewer{}, nil), "Add Brewer")
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
		renderTemplate(w, r, templates.NoBrewers())
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

	renderTemplate(w, r, templates.BrewersList(brewers), "Brewers")
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

	renderTemplate(w, r, templates.Brewer(brewer), brewer.Name)
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
		renderTemplate(w, r, templates.AddUserForm(db.User{Username: formUsername}, validationErrors))
		return
	}

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(formPassword), bcrypt.DefaultCost)
	if err != nil {
		errMsg := fmt.Sprintf("Error when hashing password: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	// Add the user to the user store
	s.userStore.AddUser(context.Background(), db.AddUserParams{
		Username:     formUsername,
		PasswordHash: string(passwordHash),
	})

	renderTemplate(w, r, templates.AddUserForm(db.User{}, nil))
}

// GET /user/add
func (s *server) getUserFormHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, templates.AddUserForm(db.User{}, nil), "Add User")
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
		renderTemplate(w, r, templates.NoUsers())
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

	renderTemplate(w, r, templates.UsersList(users), "Users")
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

	renderTemplate(w, r, templates.User(user), user.Username)
}

// POST /beer
func (s *server) addBeerHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Adding beer")
	if err := r.ParseForm(); err != nil {
		s.logger.Printf("Error when parsing form: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	formBrewerID := r.FormValue("brewer-id")
	formName := r.FormValue("name")
	formStyle := r.FormValue("style")
	formAbv := r.FormValue("abv")
	formRating := r.FormValue("rating")
	formNotes := r.FormValue("notes")

	brewers, err := s.brewerStore.GetBrewers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting brewers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	validationErrors := make(map[string]string)
	if formName == "" {
		validationErrors["name"] = "Name is required"
	}
	if formAbv == "" {
		validationErrors["abv"] = "ABV is required"
	}
	if len(validationErrors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		renderTemplate(w, r, templates.AddBeerForm(db.Beer{Name: formName}, brewers, validationErrors, false))
		return
	}

	maybeBrewerID := sql.NullInt64{}

	if formBrewerID != "" {
		brewerID, err := strconv.Atoi(formBrewerID)
		if err != nil {
			errMsg := fmt.Sprintf("Error when converting brewer id to int: %v", err)
			s.logger.Print(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
		maybeBrewerID = sql.NullInt64{Valid: true, Int64: int64(brewerID)}
	}

	rating, err := strconv.ParseFloat(formRating, 64)
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting rating to float: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	abv, err := strconv.ParseFloat(formAbv, 64)
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting ABV to float: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	beer, err := s.beerStore.AddBeer(r.Context(), db.AddBeerParams{
		BrewerID: maybeBrewerID,
		Name:     formName,
		Style:    sql.NullString{Valid: true, String: formStyle},
		Abv:      abv,
		Rating:   sql.NullFloat64{Valid: true, Float64: rating},
		Notes:    sql.NullString{Valid: true, String: formNotes},
	})
	if err != nil {
		errMsg := fmt.Sprintf("Error when adding beer: %v", err)
		s.logger.Print(errMsg)

		validationErrors := make(map[string]string)

		switch err := err.(type) {
		case store.ErrMissingField:
			validationErrors[err.Field] = "This field is required"
			w.WriteHeader(http.StatusUnprocessableEntity)
		case store.ErrBrewerNotFound:
			validationErrors["brewer-id"] = fmt.Sprintf("Brewer with id %d not found", err.ID)
			w.WriteHeader(http.StatusNotFound)
		case store.ErrBeerAlreadyExists:
			validationErrors["name"] = fmt.Sprintf("%s by brewer %d already exists", err.Name, err.BrewerId)
		default:
			http.Error(w, errMsg, http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
		}
		renderTemplate(w, r, templates.AddBeerForm(db.Beer{Name: formName}, brewers, validationErrors, false))
		return
	}

	renderTemplate(w, r, templates.AddBeerForm(db.Beer{}, brewers, nil, false))
	renderTemplate(w, r, templates.Beer(beer))
}

// PUT /beer/{id}
func (s *server) updateBeerHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	s.logger.Printf("Updating beer with id: %d", id)

	formBrewerID := r.FormValue("brewer-id")
	formName := r.FormValue("name")
	formStyle := r.FormValue("style")
	formAbv := r.FormValue("abv")
	formRating := r.FormValue("rating")
	formNotes := r.FormValue("notes")

	brewers, err := s.brewerStore.GetBrewers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting brewers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	validationErrors := make(map[string]string)
	if formName == "" {
		validationErrors["name"] = "Name is required"
	}
	if formAbv == "" {
		validationErrors["abv"] = "ABV is required"
	}
	if len(validationErrors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		renderTemplate(w, r, templates.AddBeerForm(db.Beer{Name: formName}, brewers, validationErrors, false))
		return
	}

	maybeBrewerID := sql.NullInt64{}

	if formBrewerID != "" {
		brewerID, err := strconv.Atoi(formBrewerID)
		if err != nil {
			errMsg := fmt.Sprintf("Error when converting brewer id to int: %v", err)
			s.logger.Print(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
		maybeBrewerID = sql.NullInt64{Valid: true, Int64: int64(brewerID)}
	}

	rating, err := strconv.ParseFloat(formRating, 64)
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting rating to float: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	abv, err := strconv.ParseFloat(formAbv, 64)
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting ABV to float: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	beer, err := s.beerStore.UpdateBeer(r.Context(), db.UpdateBeerParams{
		ID:       int64(id),
		BrewerID: maybeBrewerID,
		Name:     sql.NullString{Valid: true, String: formName},
		Style:    sql.NullString{Valid: true, String: formStyle},
		Abv:      sql.NullFloat64{Valid: true, Float64: abv},
		Rating:   sql.NullFloat64{Valid: true, Float64: rating},
		Notes:    sql.NullString{Valid: true, String: formNotes},
	})
	if err != nil {
		errMsg := fmt.Sprintf("Error when updating beer: %v", err)
		s.logger.Print(errMsg)

		validationErrors := make(map[string]string)

		switch err := err.(type) {
		case store.ErrMissingField:
			validationErrors[err.Field] = "This field is required"
			w.WriteHeader(http.StatusUnprocessableEntity)
		case store.ErrBrewerNotFound:
			validationErrors["brewer-id"] = fmt.Sprintf("Brewer with id %d not found", err.ID)
			w.WriteHeader(http.StatusNotFound)
		case store.ErrBeerAlreadyExists:
			validationErrors["name"] = fmt.Sprintf("%s by brewer %d already exists", err.Name, err.BrewerId)
			w.WriteHeader(http.StatusNotFound)
		default:
			http.Error(w, errMsg, http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
		}
		renderTemplate(w, r, templates.AddBeerForm(db.Beer{Name: formName}, brewers, validationErrors, true))
		return
	}

	renderTemplate(w, r, templates.Beer(beer))
}

// GET /beer/add or GET /beer/{id}/edit
func (s *server) getBeerFormHandler(w http.ResponseWriter, r *http.Request) {
	brewers, err := s.brewerStore.GetBrewers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting brewers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	if strings.Contains(r.URL.Path, "/edit") && r.PathValue("id") != "" {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
			s.logger.Print(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		beer, err := s.beerStore.GetBeer(r.Context(), int64(id))
		if err != nil {
			errMsg := fmt.Sprintf("Error when getting beer: %v", err)
			s.logger.Print(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}

		renderTemplate(w, r, templates.AddBeerForm(beer, brewers, nil, true), "Edit Beer")
		return
	}

	renderTemplate(w, r, templates.AddBeerForm(db.Beer{}, brewers, nil, false), "Add Beer")
}

// DELETE /beer/{id}
func (s *server) deleteBeerHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("Deleting beer with id: %s", r.PathValue("id"))
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	_, err = s.beerStore.DeleteBeer(r.Context(), int64(id))
	if err != nil {
		errMsg := fmt.Sprintf("Error when deleting beer: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	// Check if that was the last beer
	numBeers, err := s.beerStore.CountBeers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when counting beers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	if numBeers == 0 {
		// If we just deleted the last beer, render the no beers template
		renderTemplate(w, r, templates.NoBeers())
	} else {
		// Return nothing so the target of the delete request is replaced with nothing, i.e. removed
		w.WriteHeader(http.StatusNoContent)
	}
}

// GET /beers
func (s *server) listBeersHandler(w http.ResponseWriter, r *http.Request) {
	beers, err := s.beerStore.GetBeers(r.Context())
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting beers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	renderTemplate(w, r, templates.BeersList(beers), "Beers")
}

// POST /beer/search
func (s *server) searchBeersHandler(w http.ResponseWriter, r *http.Request) {
	// If the query is empty, just list all beers
	query := r.FormValue("q")
	if query == "" {
		http.Redirect(w, r, "/beers", http.StatusSeeOther)
		return
	}

	beers, err := s.beerStore.SearchBeers(r.Context(), sql.NullString{Valid: true, String: query})
	if err != nil {
		errMsg := fmt.Sprintf("Error when searching beers: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	renderTemplate(w, r, templates.BeersList(beers), "Beers")
}

// GET /beer/{id}
func (s *server) getBeerHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		errMsg := fmt.Sprintf("Error when converting id to int: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	beer, err := s.beerStore.GetBeer(r.Context(), int64(id))
	if err != nil {
		errMsg := fmt.Sprintf("Error when getting beer: %v", err)
		s.logger.Print(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	renderTemplate(w, r, templates.Beer(beer), beer.Name)
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
		renderTemplate(w, r, templates.LoginForm(validationErrors))
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
		renderTemplate(w, r, templates.LoginForm(validationErrors))
		return
	}

	// Check if the password is correct
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(formPassword))
	if err != nil {
		validationErrors["password"] = "Username or password is incorrect"
		w.WriteHeader(http.StatusUnauthorized)
		renderTemplate(w, r, templates.LoginForm(validationErrors))
		return
	}

	// Generate a session token
	err = s.sessionStore.WriteNew(w, r, user.ID)

	if err != nil {
		errMsg := fmt.Sprintf("Error when saving session: %v", err)
		s.logger.Print(errMsg)
		w.WriteHeader(http.StatusInternalServerError)
		validationErrors["password"] = "Internal server error"
		renderTemplate(w, r, templates.LoginForm(validationErrors))
		return
	}

	s.userStore.SetUserLastLogin(r.Context(), user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
