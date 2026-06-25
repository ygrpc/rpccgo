# rpccgofluttersharedso

Android-only Flutter UI for `examples/flutter-shared-so`.

The app starts `SharedSoRuntimeService` on launch. Kotlin buttons call the Go `.so` through the Service and JNI; Dart buttons call the same `.so` directly through generated FFI bindings.

## Dart lifecycle

`main.dart` wraps the app with generated `RpccgoLifecycleScope`:

```dart
runApp(const RpccgoLifecycleScope(child: SharedSoApp()));
```

The scope lives in `lib/gen/rpccgo.dart` and cancels registered generated Dart FFI streams when the Flutter tree is disposed or the app lifecycle reaches `detached`.

## Kotlin/JNI lifecycle

`Kotlin Start Stream` starts an Activity-owned JNI callback stream through the generated owner-aware API:

```kotlin
val stream = SharedSoDemoJni.WatchRuntimeStateStartCallback(
    this,
    request,
    listener,
)
```

The returned `RpccgoCallbackStream` can be canceled manually. If the Activity is destroyed first, the generated Kotlin/JNI wrapper cancels it automatically and suppresses callbacks after cancel.

Use `Kotlin Start Stream` with `Close Activity` to verify the stream stops without `FlutterJNI was detached from native C++` warnings.
