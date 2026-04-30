package integration

import "testing"

func TestNativeStage3Acceptance(t *testing.T) {
	t.Run("go native unary", TestNativeUnaryClientRoutesToGoNativeServer)
	t.Run("cgo native unary", TestNativeCGOServerUnaryRoutesThroughDispatcher)
	t.Run("go native client streaming", TestNativeClientStreamingRoutesToGoNativeServer)
	t.Run("cgo native client streaming", TestNativeClientStreamingRoutesToCGONativeServer)
	t.Run("go native server streaming", TestNativeServerStreamingRoutesToGoNativeServer)
	t.Run("cgo native server streaming", TestNativeServerStreamingRoutesToCGONativeServer)
	t.Run("go native bidi streaming", TestNativeBidiStreamingRoutesToGoNativeServer)
	t.Run("cgo native bidi streaming", TestNativeBidiStreamingRoutesToCGONativeServer)
}
