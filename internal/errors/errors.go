/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errors

import (
	"errors"
	"fmt"
)

// TransientError wraps errors that are temporary and worth retrying.
// The circuit breaker should count these normally.
type TransientError struct {
	Err error
}

func (e *TransientError) Error() string { return e.Err.Error() }
func (e *TransientError) Unwrap() error { return e.Err }

// NewTransient wraps an error as transient.
func NewTransient(err error) error {
	if err == nil {
		return nil
	}
	return &TransientError{Err: err}
}

// ValidationError represents a permanent configuration or spec validation error.
// These errors will never self-heal, so the circuit breaker should NOT count them
// toward the failure threshold.
type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

// NewValidation wraps an error as a validation error.
func NewValidation(err error) error {
	if err == nil {
		return nil
	}
	return &ValidationError{Err: err}
}

// NewValidationf creates a new ValidationError with a formatted message.
func NewValidationf(format string, args ...any) error {
	return &ValidationError{Err: fmt.Errorf(format, args...)}
}

// IsValidation checks if an error is a ValidationError.
func IsValidation(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
