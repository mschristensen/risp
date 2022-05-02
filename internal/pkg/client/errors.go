package client

import "github.com/pkg/errors"

// ErrNotDone indicates that the client is not in the done state.
var ErrNotDone = errors.New("not done")

// ErrMissingChecksum indicates that the checksum is missing.
var ErrMissingChecksum = errors.New("missing checksum")

// ErrChecksumMismatch indicates that the checksum does not match the expected value.
var ErrChecksumMismatch = errors.New("checksum mismatch")

// ErrClientDisconnected indicates that the client disconnected from the server but should reconnect.
var ErrClientDisconnected = errors.New("client disconnected")
