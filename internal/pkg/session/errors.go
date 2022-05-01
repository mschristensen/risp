package session

import "errors"

var ErrSessionNotFound = errors.New("session not found")
var ErrSessionAlreadyExists = errors.New("session already exists")
