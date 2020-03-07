package jobs

import (
	"fmt"
)

const ErrQueueIsEmpty = "QueueIsEmpty"

// Error wraps lower level errors with code, message and an original error
type QueueError struct {
	Code    string
	Message string
	OrigErr error
}

// New constructs a QueueError and returns it as an error
func NewQueueError(code, message string, err error) QueueError {
	return QueueError{
		Code:    code,
		Message: message,
		OrigErr: err,
	}
}

// Error Satisfies the Error interface
func (e QueueError) Error() string {
	return e.String()
}

// String returns the error as string
func (e QueueError) String() string {
	if e.OrigErr != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.OrigErr)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the contained error
func (e QueueError) Unwrap() error {
	return e.OrigErr
}
