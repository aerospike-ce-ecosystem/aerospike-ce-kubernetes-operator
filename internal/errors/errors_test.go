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
	"testing"
)

func TestNewValidationf(t *testing.T) {
	err := NewValidationf("template %q not found", "my-tmpl")

	expected := `template "my-tmpl" not found`
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("errors.As should match *ValidationError")
	}
}

func TestIsValidation(t *testing.T) {
	t.Run("true for ValidationError", func(t *testing.T) {
		err := NewValidationf("bad config")
		if !IsValidation(err) {
			t.Fatal("expected true")
		}
	})

	t.Run("true for wrapped ValidationError", func(t *testing.T) {
		inner := NewValidationf("bad config")
		wrapped := fmt.Errorf("outer: %w", inner)
		if !IsValidation(wrapped) {
			t.Fatal("expected true through wrapping chain")
		}
	})

	t.Run("false for non-validation error", func(t *testing.T) {
		err := fmt.Errorf("some transient error")
		if IsValidation(err) {
			t.Fatal("expected false")
		}
	})

	t.Run("false for nil", func(t *testing.T) {
		if IsValidation(nil) {
			t.Fatal("expected false for nil")
		}
	})
}

func TestErrorsIs_WrappingChain(t *testing.T) {
	sentinel := fmt.Errorf("sentinel")
	validated := NewValidationf("wrapping: %v", sentinel)
	wrapped := fmt.Errorf("layer1: %w", validated)
	doubleWrapped := fmt.Errorf("layer2: %w", wrapped)

	if !IsValidation(doubleWrapped) {
		t.Fatal("IsValidation should find ValidationError through double wrapping")
	}
}
