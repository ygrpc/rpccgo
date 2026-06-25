# rpccgofluttersharedso

Android-only Flutter UI for `examples/flutter-shared-so`.

The app starts `SharedSoRuntimeService` on launch. Kotlin buttons call the Go `.so` through the Service and JNI; Dart buttons call the same `.so` directly through generated FFI bindings.

`main.dart` wraps the app with generated `RpccgoLifecycleScope`:

```dart
runApp(const RpccgoLifecycleScope(child: SharedSoApp()));
```

The scope lives in `lib/gen/rpccgo.dart` and cancels registered generated Dart FFI streams when the Flutter tree is disposed or the app lifecycle reaches `detached`.
