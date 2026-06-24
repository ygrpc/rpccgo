# rpccgofluttersharedso

Android-only Flutter UI for `examples/flutter-shared-so`.

The app starts `SharedSoRuntimeService` on launch. Kotlin buttons call the Go `.so` through the Service and JNI; Dart buttons call the same `.so` directly through generated FFI bindings.
