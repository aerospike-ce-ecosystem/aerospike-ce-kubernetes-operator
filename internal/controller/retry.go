package controller

// retryOnTransient calls fn once. If it returns a transient Aerospike error
// (connection reset, timeout, etc.), fn is retried exactly once. On the second
// failure the retry error is returned. Non-transient errors are returned
// immediately without retry.
func retryOnTransient(fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}
	if !isTransientAeroError(err) {
		return err
	}
	return fn()
}
