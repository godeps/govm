package client

import "errors"

var (
	// ErrNativeUnavailable means govm is running with the non-native stub binding.
	ErrNativeUnavailable = errors.New("govm native bridge unavailable: rebuild with -tags govm_native and bundled native libs")
)
