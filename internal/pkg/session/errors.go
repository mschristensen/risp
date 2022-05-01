package session

import "errors"

// ErrSessionNotFound indicates that the session was not found for this uuid.
var ErrSessionNotFound = errors.New("session not found")

// ErrSessionAlreadyExists indicates that the session already exists for this uuid.
var ErrSessionAlreadyExists = errors.New("session already exists")
