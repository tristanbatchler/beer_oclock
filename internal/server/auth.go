package server

import (
	"beer_oclock/internal/store/users"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

type BeerOclockSessionStore struct {
	sessionStore sessions.Store
	userStore    *users.UserStore
	logger       *log.Logger
}

func NewBeerOclockSessionStore(sessionStore sessions.Store, userStore *users.UserStore) *BeerOclockSessionStore {
	return &BeerOclockSessionStore{
		sessionStore: sessionStore,
		userStore:    userStore,
		logger:       log.New(os.Stdout, "[Session Store]: ", log.LstdFlags),
	}
}

func (s *BeerOclockSessionStore) ValidateSession(r *http.Request) (int64, error) {
	// Get the authentication cookie
	session, err := s.sessionStore.Get(r, "session")
	if err != nil {
		return 0, fmt.Errorf("Error when getting session (it was nil): %v", err)

	}

	userIdValue := session.Values["userId"]
	userId, ok := userIdValue.(int64)
	if !ok {
		return 0, fmt.Errorf("Invalid user ID in session (could not cast to int64): %v", userIdValue)
	}

	// Validate the user exists
	_, err = s.userStore.GetUserById(r.Context(), userId)
	if err != nil {
		return 0, fmt.Errorf("Error when validating userId in session (could not get user by ID): %v", err)
	}

	return userId, nil
}

func (s *BeerOclockSessionStore) WriteNew(w http.ResponseWriter, r *http.Request, userId int64) error {
	session, err := s.sessionStore.New(r, "session")
	if err != nil {
		return err
	}
	session.Options = &sessions.Options{
		MaxAge:   3600, // 1 hour
		HttpOnly: true, // JS cannot access the cookie
	}

	session.Values["userId"] = userId
	return session.Save(r, w)
}

func (s *BeerOclockSessionStore) EraseCurrent(w http.ResponseWriter, r *http.Request) {
	session, err := s.sessionStore.Get(r, "session")
	if err != nil {
		// Maybe an overreaction to log this as fatal, but it's important to know if this happens
		s.logger.Fatalf("Error when getting session so can't invalidate it: %v", err)
	}

	session.Options.MaxAge = -1

	// Write back the invalidated session
	err = session.Save(r, w)
}
