package state

import "errors"

var (
	// ErrStateNotFound indicates the state file does not exist
	ErrStateNotFound = errors.New("state file not found")

	// ErrStateCorrupted indicates the state file contains invalid JSON
	ErrStateCorrupted = errors.New("state file is corrupted")

	// ErrVersionMismatch indicates an incompatible state schema version
	ErrVersionMismatch = errors.New("state version mismatch")
)
