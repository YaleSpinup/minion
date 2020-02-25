package apierror

import (
	"fmt"
)

// ErrBadRequest indicates a bad request or input
const ErrBadRequest = "BadRequest"

// ErrForbidden indicates a lack of permissions for the given resource
const ErrForbidden = "Forbidden"

// ErrNotFound indicates a the requested object is missing/not found
const ErrNotFound = "NotFound"

// ErrConflict indicates a conflict with an existing resource
const ErrConflict = "Conflict"

// ErrLimitExceeded indicates a service or rate limit has been exceeded
const ErrLimitExceeded = "LimitExceeded"

// ErrServiceUnavailable indicates an internal or external service is not available
const ErrServiceUnavailable = "ServiceUnavailable"

// ErrInternalError indicates an unknown internal error occurred
const ErrInternalError = "InternalError"

// Error wraps lower level errors with code, message and an original error.  This is
// modelled after the awserr with the intention of standardizing the output.
type Error struct {
	Code    string
	Message string
	OrigErr error
}

// New constructs an Error and returns it as an error
func New(code, message string, err error) Error {
	return Error{
		Code:    code,
		Message: message,
		OrigErr: err,
	}
}

// Error Satisfies the Error interface
func (e Error) Error() string {
	return e.String()
}

// String returns the error as string
func (e Error) String() string {
	if e.OrigErr != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.OrigErr)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the contained error
func (e Error) Unwrap() error {
	return e.OrigErr
}
