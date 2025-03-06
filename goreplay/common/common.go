package common

import "errors"

// Message represents data across plugins
type Message struct {
	Meta []byte // metadata
	Data []byte // actual data
}

// ErrorStopped is the error returned when the go routines reading the input is stopped.
var ErrorStopped = errors.New("reading stopped")
