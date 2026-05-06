package integration

import "testing"

func TestConverterSnapshotAndCancel(t *testing.T) {
	t.Run("stream start captures cgo native snapshot", func(t *testing.T) {
		runMessageDirectPathFixture(t, "TestConverterStreamStartCapturesCGONativeSnapshot")
	})
	t.Run("cancel propagates and finalizes handle", func(t *testing.T) {
		runMessageDirectPathFixture(t, "TestConverterCancelPropagatesToCGONativeAndFinalizesHandle")
	})
}
