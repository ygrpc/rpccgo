package integration

import "testing"

func TestStage4BMessageClientToGoNativeServerRoutesThroughConverter(t *testing.T) {
	t.Run("unary and streaming message client entries", func(t *testing.T) {
		runMessageDirectPathFixture(t, "TestMessageContractMismatch")
	})
}

func TestMessageClientToGoNative(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageContractMismatch")
}

func TestStage4BMessageClientToCGONativeServerRoutesThroughConverter(t *testing.T) {
	t.Run("unary and streaming message client entries", func(t *testing.T) {
		runMessageDirectPathFixture(t, "TestMessageClientToCGONative")
	})
}

func TestMessageClientToCGONative(t *testing.T) {
	runMessageDirectPathFixture(t, "TestMessageClientToCGONative")
}

func TestStage4BNativeClientToCGOMessageServerRoutesThroughConverter(t *testing.T) {
	t.Run("unary and streaming native client entries", func(t *testing.T) {
		runMessageDirectPathFixture(t, "TestNativeContractMismatch")
	})
}

func TestNativeClientToCGOMessage(t *testing.T) {
	runMessageDirectPathFixture(t, "TestNativeContractMismatch")
}

func TestStage4BConverterFixtureShape(t *testing.T) {
	t.Log("fixture: generate @rpccgo: native service with unary, client streaming, server streaming, and bidi methods")
	t.Log("fixture: write generated runtime, native server, cgo native client, cgo message client, and cgo message server files into a temporary module")
	t.Log("helper reuse: runMessageDirectPathFixture builds the module and runs one generated cgo package test by name")
	t.Log("current assertion: mismatch calls route through dispatcher snapshots and generated codec wrappers")
	t.Log("covered paths: cgo message client to Go native server, cgo message client to cgo native server, and cgo native client to cgo message server")
	t.Log("pending broader payload coverage: add non-empty scalar/string/bytes/repeated fixture messages")
}
