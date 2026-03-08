package controller

import (
	"errors"
	"testing"
)

func TestRetryOnTransient_Success(t *testing.T) {
	calls := 0
	err := retryOnTransient(func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryOnTransient_PermanentError(t *testing.T) {
	calls := 0
	permanent := errors.New("invalid privilege")
	err := retryOnTransient(func() error {
		calls++
		return permanent
	})
	if err != permanent {
		t.Fatalf("expected permanent error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call (no retry for permanent error), got %d", calls)
	}
}

func TestRetryOnTransient_TransientThenSuccess(t *testing.T) {
	calls := 0
	err := retryOnTransient(func() error {
		calls++
		if calls == 1 {
			return errors.New("connection reset by peer")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error after retry, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestRetryOnTransient_TransientThenTransient(t *testing.T) {
	calls := 0
	err := retryOnTransient(func() error {
		calls++
		return errors.New("connection reset by peer")
	})
	if err == nil {
		t.Fatal("expected error after both calls fail")
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestRetryOnTransient_TransientThenPermanent(t *testing.T) {
	calls := 0
	err := retryOnTransient(func() error {
		calls++
		if calls == 1 {
			return errors.New("read tcp 10.0.0.1:3000: timeout")
		}
		return errors.New("invalid privilege")
	})
	if err == nil {
		t.Fatal("expected error after retry returns permanent error")
	}
	if err.Error() != "invalid privilege" {
		t.Fatalf("expected permanent error from retry, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}
