package rpcruntime

import "errors"

// NativeReleaser is implemented by native wrappers that release owned resources.
type NativeReleaser interface {
	Release() error
}

// NativeReleaseStack releases native call-scoped resources in reverse acquisition order.
type NativeReleaseStack []NativeReleaser

// Release releases all resources in reverse order and joins release errors.
func (s NativeReleaseStack) Release() error {
	var err error
	for i := len(s) - 1; i >= 0; i-- {
		err = errors.Join(err, s[i].Release())
	}
	return err
}
