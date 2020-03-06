package jobs

import (
	"context"
	"fmt"
)

const ErrMissingDetails = "MissingDetails"
const ErrPreExecFailure = "PreExecutionFailure"
const ErrExecFailure = "ExecutionFailure"
const ErrPostExecFailure = "PostExecutionFailure"

// Runner has a Run method and runs a job
type Runner interface {
	Run(ctx context.Context, account string, parameters interface{}) (string, error)
}

type RunnerError struct {
	Code    string
	Message string
	OrigErr error
}

// New constructs a RunnerError and returns it as an error
func NewRunnerError(code, message string, err error) RunnerError {
	return RunnerError{
		Code:    code,
		Message: message,
		OrigErr: err,
	}
}

// Error Satisfies the Error interface
func (e RunnerError) Error() string {
	return e.String()
}

// String returns the error as string
func (e RunnerError) String() string {
	if e.OrigErr != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.OrigErr)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the contained error
func (e RunnerError) Unwrap() error {
	return e.OrigErr
}
