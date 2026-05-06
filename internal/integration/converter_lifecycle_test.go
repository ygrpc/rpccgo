package integration

import "testing"

func TestConverterLifecycleErrorPrecedence(t *testing.T) {
	t.Run("converter error does not call cgo native server", func(t *testing.T) {
		runMessageDirectPathFixture(t, "TestConverterErrorDoesNotCallCGONativeServer")
	})
	t.Run("downstream cgo native error is not covered by converter", func(t *testing.T) {
		runMessageDirectPathFixture(t, "TestDownstreamCGONativeErrorIsNotCoveredByConverter")
	})
}
