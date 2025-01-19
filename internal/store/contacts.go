// Description: This file contains the contactStore struct and its methods
// that are used to store and retrieve contacts from the store.
// This is for example purposes only and just uses an in-memory Map
package store

import (
	"encoding/json"
	"fmt"
	"log"
)

type Contact struct {
	Id    int
	Name  string
	Email string
}

type contactStore struct {
	logger   *log.Logger
	contacts map[int]Contact
}

func NewContactStore(logger *log.Logger) *contactStore {
	return &contactStore{
		logger:   logger,
		contacts: make(map[int]Contact),
	}
}

func (cs *contactStore) AddContact(contact Contact) error {
	if contact.Email == "" {
		return fmt.Errorf("email is required")
	}

	// check if contact already exists
	if _, ok := cs.contacts[contact.Id]; ok {
		return fmt.Errorf("contact already exists")
	}

	// add contact to map
	cs.contacts[contact.Id] = contact
	cs.logger.Printf("contact added: %v", contact)
	return nil
}

func (cs *contactStore) GetContacts() ([]Contact, error) {
	if cs.contacts == nil {
		return nil, fmt.Errorf("no contacts found")
	}

	// create contacts slice
	contacts := make([]Contact, 0, len(cs.contacts))
	for _, contact := range cs.contacts {
		contacts = append(contacts, contact)
	}
	return contacts, nil
}

func (cs *contactStore) DeleteContact(id int) error {
	if _, ok := cs.contacts[id]; !ok {
		return fmt.Errorf("contact not found")
	}
	delete(cs.contacts, id)
	return nil
}

func DecodeContact(payload []byte) (Contact, error) {
	var contact Contact
	if err := json.Unmarshal(payload, &contact); err != nil {
		return Contact{}, fmt.Errorf("error decoding contact: %w", err)
	}
	return contact, nil
}
