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

func TestNewTransient(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if got := NewTransient(nil); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("wraps and unwraps correctly", func(t *testing.T) {
		inner := fmt.Errorf("connection refused")
		wrapped := NewTransient(inner)

		if wrapped.Error() != "connection refused" {
			t.Fatalf("unexpected message: %s", wrapped.Error())
		}

		var te *TransientError
		if !errors.As(wrapped, &te) {
			t.Fatal("errors.As should match *TransientError")
		}
		if te.Err != inner {
			t.Fatal("unwrapped error should be the original")
		}
		if !errors.Is(wrapped, inner) {
			t.Fatal("errors.Is should find the inner error")
		}
	})
}

func TestNewValidation(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		if got := NewValidation(nil); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("wraps and unwraps correctly", func(t *testing.T) {
		inner := fmt.Errorf("invalid size")
		wrapped := NewValidation(inner)

		if wrapped.Error() != "invalid size" {
			t.Fatalf("unexpected message: %s", wrapped.Error())
		}

		var ve *ValidationError
		if !errors.As(wrapped, &ve) {
			t.Fatal("errors.As should match *ValidationError")
		}
		if ve.Err != inner {
			t.Fatal("unwrapped error should be the original")
		}
		if !errors.Is(wrapped, inner) {
			t.Fatal("errors.Is should find the inner error")
		}
	})
}

func TestNewValidationf(t *testing.T) {
	err := NewValidationf("template %q not found in namespace %q", "my-tmpl", "default")

	expected := `template "my-tmpl" not found in namespace "default"`
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
		err := NewValidation(fmt.Errorf("bad config"))
		if !IsValidation(err) {
			t.Fatal("expected true")
		}
	})

	t.Run("true for wrapped ValidationError", func(t *testing.T) {
		inner := NewValidation(fmt.Errorf("bad config"))
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

	t.Run("false for TransientError", func(t *testing.T) {
		err := NewTransient(fmt.Errorf("timeout"))
		if IsValidation(err) {
			t.Fatal("expected false for TransientError")
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
	validated := NewValidation(sentinel)
	wrapped := fmt.Errorf("layer1: %w", validated)
	doubleWrapped := fmt.Errorf("layer2: %w", wrapped)

	if !errors.Is(doubleWrapped, sentinel) {
		t.Fatal("errors.Is should find sentinel through double wrapping")
	}
	if !IsValidation(doubleWrapped) {
		t.Fatal("IsValidation should find ValidationError through double wrapping")
	}
}
