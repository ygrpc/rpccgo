/// Package-level library URI reserved as the shared native-asset ID for rpccgo.
///
/// Future `protoc-gen-rpc-cgo-dart` output is expected to bind `@Native`
/// symbols against `package:rpccgo_connect_greeter_dart_client/rpccgo.dart`
/// explicitly; importing or re-exporting this library does not change the
/// default asset ID of declarations emitted into another library URI.
library rpccgo_connect_greeter_dart_client;

export 'gen/greeter.greeter.rpccgo.dart';
export 'gen/greeter.pb.dart';
