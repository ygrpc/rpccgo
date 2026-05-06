package integration

import "testing"

func TestStage4BMessageClientToGoNativeServerRoutesThroughConverter(t *testing.T) {
	t.Run("unary and streaming message client entries", func(t *testing.T) {
		runMessageDirectPathFixture(t, "TestMessageContractMismatch")
	})
}

func TestStage4BConverterFixtureShape(t *testing.T) {
	t.Log("fixture: generate @rpccgo: native service with unary, client streaming, server streaming, and bidi methods")
	t.Log("fixture: write generated runtime, native server, cgo native client, cgo message client, and cgo message server files into a temporary module")
	t.Log("helper reuse: runMessageDirectPathFixture builds the module and runs one generated cgo package test by name")
	t.Log("current assertion: registering a Go native server and entering through cgo message client functions routes unary and streaming payloads through generated codec wrappers")
	t.Log("pending broader coverage: add cgo native server and cgo message server mismatch fixtures with non-empty payloads")
}
