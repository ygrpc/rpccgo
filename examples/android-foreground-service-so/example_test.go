package androidforegroundserviceso

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/rpccgo-android-foreground-service-so/internal/backend"
	foregroundservicev1 "example.com/rpccgo-android-foreground-service-so/proto"
)

func TestForegroundServiceDemoMessageContract(t *testing.T) {
	server := backend.NewForegroundServiceDemoServer()
	if err := foregroundservicev1.RegisterForegroundServiceDemoConnectHandler(server); err != nil {
		t.Fatalf("register demo server: %v", err)
	}
	defer func() {
		if err := foregroundservicev1.ClearForegroundServiceDemoServer(); err != nil {
			t.Fatalf("clear demo server: %v", err)
		}
	}()

	info, err := foregroundservicev1.InvokeForegroundServiceDemoMessageServiceInfo(context.Background(), &foregroundservicev1.ServiceInfoRequest{Caller: "test"})
	if err != nil {
		t.Fatalf("service info: %v", err)
	}
	if got, want := info.GetLibrary(), "librpccgo_android_foreground_service.so"; got != want {
		t.Fatalf("library = %q, want %q", got, want)
	}
	if info.GetPid() <= 0 || info.GetInstanceAddress() == "" {
		t.Fatalf("invalid service info: %+v", info)
	}

	ctx, cancel := context.WithCancel(context.Background())
	handle, err := foregroundservicev1.ForegroundServiceDemoMessageWatchTicksStart(ctx, &foregroundservicev1.WatchTicksRequest{
		Caller:         "go-test",
		IntervalMillis: 1,
	})
	if err != nil {
		t.Fatalf("start watch ticks: %v", err)
	}
	for i := 0; i < 2; i++ {
		tick, err := foregroundservicev1.ForegroundServiceDemoMessageWatchTicksRecv(ctx, handle)
		if err != nil {
			t.Fatalf("recv tick %d: %v", i, err)
		}
		if got, want := tick.GetCaller(), "go-test"; got != want {
			t.Fatalf("caller = %q, want %q", got, want)
		}
	}
	cancel()
	if err := foregroundservicev1.ForegroundServiceDemoMessageWatchTicksCancel(context.Background(), handle); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
		t.Fatalf("cancel watch ticks: %v", err)
	}
}

func TestAndroidProjectContracts(t *testing.T) {
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/StreamForegroundService.kt", "object : ForegroundServiceDemoJni.ForegroundServiceDemoWatchTicksListener")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/StreamForegroundService.kt", "WatchTicksStartCallback")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/StreamForegroundService.kt", "WatchTicksCancelCallback")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/StreamForegroundService.kt", "bad ui callback failed")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/StreamForegroundService.kt", "startUiUpdatingStream")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/StreamForegroundService.kt", "viaServiceRequest=")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/MainActivity.kt", "finish()")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/MainActivity.kt", "ActivityUiBridge")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/MainActivity.kt", "captured activity is not alive")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/MainActivity.kt", "bindService")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/MainActivity.kt", "activity onDestroy")
	assertFileContains(t, "android_app/app/src/main/kotlin/com/ygrpc/examples/rpccgoandroidforegroundservice/StreamForegroundService.kt", "task removed; service keeps callback stream until Stop")
	assertFileContains(t, "android_app/app/src/main/AndroidManifest.xml", "android.permission.FOREGROUND_SERVICE")
	assertFileContains(t, "android_app/app/src/main/cpp/CMakeLists.txt", "rpccgo_android_foreground_service_jni")
	assertFileContains(t, "android_app/tool/build_android_so.sh", "go build -buildmode=c-shared")
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q", path, want)
	}
}

func TestWatchTicksDefaultIntervalDoesNotBusyLoop(t *testing.T) {
	server := backend.NewForegroundServiceDemoServer()
	if err := foregroundservicev1.RegisterForegroundServiceDemoConnectHandler(server); err != nil {
		t.Fatalf("register demo server: %v", err)
	}
	defer func() {
		if err := foregroundservicev1.ClearForegroundServiceDemoServer(); err != nil {
			t.Fatalf("clear demo server: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	handle, err := foregroundservicev1.ForegroundServiceDemoMessageWatchTicksStart(ctx, &foregroundservicev1.WatchTicksRequest{})
	if err != nil {
		t.Fatalf("start watch ticks: %v", err)
	}
	if _, err := foregroundservicev1.ForegroundServiceDemoMessageWatchTicksRecv(ctx, handle); err != nil {
		t.Fatalf("recv default interval tick: %v", err)
	}
	if err := foregroundservicev1.ForegroundServiceDemoMessageWatchTicksCancel(context.Background(), handle); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
		t.Fatalf("cancel watch ticks: %v", err)
	}
}
