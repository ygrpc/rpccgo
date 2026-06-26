// This is a generated file - do not edit.
//
// Generated from shared_so.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// ComposeGreetingRequest carries the caller label and greeting target.
class ComposeGreetingRequest extends $pb.GeneratedMessage {
  factory ComposeGreetingRequest({
    $core.String? name,
    $core.String? caller,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (caller != null) result.caller = caller;
    return result;
  }

  ComposeGreetingRequest._();

  factory ComposeGreetingRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ComposeGreetingRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ComposeGreetingRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.flutter.sharedso.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'caller')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ComposeGreetingRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ComposeGreetingRequest copyWith(
          void Function(ComposeGreetingRequest) updates) =>
      super.copyWith((message) => updates(message as ComposeGreetingRequest))
          as ComposeGreetingRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ComposeGreetingRequest create() => ComposeGreetingRequest._();
  @$core.override
  ComposeGreetingRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ComposeGreetingRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ComposeGreetingRequest>(create);
  static ComposeGreetingRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get caller => $_getSZ(1);
  @$pb.TagNumber(2)
  set caller($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCaller() => $_has(1);
  @$pb.TagNumber(2)
  void clearCaller() => $_clearField(2);
}

/// ComposeGreetingResponse carries the rendered greeting and runtime markers.
class ComposeGreetingResponse extends $pb.GeneratedMessage {
  factory ComposeGreetingResponse({
    $core.String? message,
    $core.String? servedBy,
    $core.String? library,
  }) {
    final result = create();
    if (message != null) result.message = message;
    if (servedBy != null) result.servedBy = servedBy;
    if (library != null) result.library = library;
    return result;
  }

  ComposeGreetingResponse._();

  factory ComposeGreetingResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ComposeGreetingResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ComposeGreetingResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.flutter.sharedso.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'message')
    ..aOS(2, _omitFieldNames ? '' : 'servedBy')
    ..aOS(3, _omitFieldNames ? '' : 'library')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ComposeGreetingResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ComposeGreetingResponse copyWith(
          void Function(ComposeGreetingResponse) updates) =>
      super.copyWith((message) => updates(message as ComposeGreetingResponse))
          as ComposeGreetingResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ComposeGreetingResponse create() => ComposeGreetingResponse._();
  @$core.override
  ComposeGreetingResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ComposeGreetingResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ComposeGreetingResponse>(create);
  static ComposeGreetingResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get message => $_getSZ(0);
  @$pb.TagNumber(1)
  set message($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMessage() => $_has(0);
  @$pb.TagNumber(1)
  void clearMessage() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get servedBy => $_getSZ(1);
  @$pb.TagNumber(2)
  set servedBy($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasServedBy() => $_has(1);
  @$pb.TagNumber(2)
  void clearServedBy() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get library => $_getSZ(2);
  @$pb.TagNumber(3)
  set library($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLibrary() => $_has(2);
  @$pb.TagNumber(3)
  void clearLibrary() => $_clearField(3);
}

/// IncrementRuntimeStateRequest changes mutable state inside the Go runtime.
class IncrementRuntimeStateRequest extends $pb.GeneratedMessage {
  factory IncrementRuntimeStateRequest({
    $core.int? delta,
    $core.String? caller,
  }) {
    final result = create();
    if (delta != null) result.delta = delta;
    if (caller != null) result.caller = caller;
    return result;
  }

  IncrementRuntimeStateRequest._();

  factory IncrementRuntimeStateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory IncrementRuntimeStateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'IncrementRuntimeStateRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.flutter.sharedso.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'delta')
    ..aOS(2, _omitFieldNames ? '' : 'caller')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IncrementRuntimeStateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IncrementRuntimeStateRequest copyWith(
          void Function(IncrementRuntimeStateRequest) updates) =>
      super.copyWith(
              (message) => updates(message as IncrementRuntimeStateRequest))
          as IncrementRuntimeStateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static IncrementRuntimeStateRequest create() =>
      IncrementRuntimeStateRequest._();
  @$core.override
  IncrementRuntimeStateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static IncrementRuntimeStateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<IncrementRuntimeStateRequest>(create);
  static IncrementRuntimeStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get delta => $_getIZ(0);
  @$pb.TagNumber(1)
  set delta($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDelta() => $_has(0);
  @$pb.TagNumber(1)
  void clearDelta() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get caller => $_getSZ(1);
  @$pb.TagNumber(2)
  set caller($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCaller() => $_has(1);
  @$pb.TagNumber(2)
  void clearCaller() => $_clearField(2);
}

/// ReadRuntimeStateRequest identifies the path observing the Go runtime state.
class ReadRuntimeStateRequest extends $pb.GeneratedMessage {
  factory ReadRuntimeStateRequest({
    $core.String? caller,
  }) {
    final result = create();
    if (caller != null) result.caller = caller;
    return result;
  }

  ReadRuntimeStateRequest._();

  factory ReadRuntimeStateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReadRuntimeStateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReadRuntimeStateRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.flutter.sharedso.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'caller')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadRuntimeStateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadRuntimeStateRequest copyWith(
          void Function(ReadRuntimeStateRequest) updates) =>
      super.copyWith((message) => updates(message as ReadRuntimeStateRequest))
          as ReadRuntimeStateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReadRuntimeStateRequest create() => ReadRuntimeStateRequest._();
  @$core.override
  ReadRuntimeStateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ReadRuntimeStateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReadRuntimeStateRequest>(create);
  static ReadRuntimeStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get caller => $_getSZ(0);
  @$pb.TagNumber(1)
  set caller($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCaller() => $_has(0);
  @$pb.TagNumber(1)
  void clearCaller() => $_clearField(1);
}

/// RuntimeStateResponse identifies one mutable Go runtime state instance.
class RuntimeStateResponse extends $pb.GeneratedMessage {
  factory RuntimeStateResponse({
    $fixnum.Int64? value,
    $fixnum.Int64? revision,
    $core.String? instanceAddress,
    $core.String? caller,
    $core.int? pid,
  }) {
    final result = create();
    if (value != null) result.value = value;
    if (revision != null) result.revision = revision;
    if (instanceAddress != null) result.instanceAddress = instanceAddress;
    if (caller != null) result.caller = caller;
    if (pid != null) result.pid = pid;
    return result;
  }

  RuntimeStateResponse._();

  factory RuntimeStateResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RuntimeStateResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RuntimeStateResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.flutter.sharedso.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'value')
    ..aInt64(2, _omitFieldNames ? '' : 'revision')
    ..aOS(3, _omitFieldNames ? '' : 'instanceAddress')
    ..aOS(4, _omitFieldNames ? '' : 'caller')
    ..aI(5, _omitFieldNames ? '' : 'pid')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeStateResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeStateResponse copyWith(void Function(RuntimeStateResponse) updates) =>
      super.copyWith((message) => updates(message as RuntimeStateResponse))
          as RuntimeStateResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RuntimeStateResponse create() => RuntimeStateResponse._();
  @$core.override
  RuntimeStateResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RuntimeStateResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RuntimeStateResponse>(create);
  static RuntimeStateResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get value => $_getI64(0);
  @$pb.TagNumber(1)
  set value($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasValue() => $_has(0);
  @$pb.TagNumber(1)
  void clearValue() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get revision => $_getI64(1);
  @$pb.TagNumber(2)
  set revision($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRevision() => $_has(1);
  @$pb.TagNumber(2)
  void clearRevision() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get instanceAddress => $_getSZ(2);
  @$pb.TagNumber(3)
  set instanceAddress($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasInstanceAddress() => $_has(2);
  @$pb.TagNumber(3)
  void clearInstanceAddress() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get caller => $_getSZ(3);
  @$pb.TagNumber(4)
  set caller($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCaller() => $_has(3);
  @$pb.TagNumber(4)
  void clearCaller() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get pid => $_getIZ(4);
  @$pb.TagNumber(5)
  set pid($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPid() => $_has(4);
  @$pb.TagNumber(5)
  void clearPid() => $_clearField(5);
}

/// SetTorchRequest asks Android to enable or disable the device torch.
class SetTorchRequest extends $pb.GeneratedMessage {
  factory SetTorchRequest({
    $core.bool? enabled,
    $core.String? caller,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    if (caller != null) result.caller = caller;
    return result;
  }

  SetTorchRequest._();

  factory SetTorchRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetTorchRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetTorchRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.flutter.sharedso.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..aOS(2, _omitFieldNames ? '' : 'caller')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetTorchRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetTorchRequest copyWith(void Function(SetTorchRequest) updates) =>
      super.copyWith((message) => updates(message as SetTorchRequest))
          as SetTorchRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetTorchRequest create() => SetTorchRequest._();
  @$core.override
  SetTorchRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetTorchRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetTorchRequest>(create);
  static SetTorchRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get caller => $_getSZ(1);
  @$pb.TagNumber(2)
  set caller($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCaller() => $_has(1);
  @$pb.TagNumber(2)
  void clearCaller() => $_clearField(2);
}

/// SetTorchResponse reports the Android torch operation result.
class SetTorchResponse extends $pb.GeneratedMessage {
  factory SetTorchResponse({
    $core.bool? enabled,
    $core.String? cameraId,
    $core.String? caller,
    $core.String? status,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    if (cameraId != null) result.cameraId = cameraId;
    if (caller != null) result.caller = caller;
    if (status != null) result.status = status;
    return result;
  }

  SetTorchResponse._();

  factory SetTorchResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetTorchResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetTorchResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.flutter.sharedso.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..aOS(2, _omitFieldNames ? '' : 'cameraId')
    ..aOS(3, _omitFieldNames ? '' : 'caller')
    ..aOS(4, _omitFieldNames ? '' : 'status')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetTorchResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetTorchResponse copyWith(void Function(SetTorchResponse) updates) =>
      super.copyWith((message) => updates(message as SetTorchResponse))
          as SetTorchResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetTorchResponse create() => SetTorchResponse._();
  @$core.override
  SetTorchResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetTorchResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetTorchResponse>(create);
  static SetTorchResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get cameraId => $_getSZ(1);
  @$pb.TagNumber(2)
  set cameraId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCameraId() => $_has(1);
  @$pb.TagNumber(2)
  void clearCameraId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get caller => $_getSZ(2);
  @$pb.TagNumber(3)
  set caller($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCaller() => $_has(2);
  @$pb.TagNumber(3)
  void clearCaller() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get status => $_getSZ(3);
  @$pb.TagNumber(4)
  set status($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStatus() => $_has(3);
  @$pb.TagNumber(4)
  void clearStatus() => $_clearField(4);
}

/// SharedSoDemo exposes one unary RPC for the shared-library validation example.
/// @rpccgo: msg-connect
class SharedSoDemoApi {
  final $pb.RpcClient _client;

  SharedSoDemoApi(this._client);

  /// ComposeGreeting returns a greeting that identifies the caller path.
  $async.Future<ComposeGreetingResponse> composeGreeting(
          $pb.ClientContext? ctx, ComposeGreetingRequest request) =>
      _client.invoke<ComposeGreetingResponse>(ctx, 'SharedSoDemo',
          'ComposeGreeting', request, ComposeGreetingResponse());

  /// IncrementRuntimeState modifies state through the Flutter FFI path.
  $async.Future<RuntimeStateResponse> incrementRuntimeState(
          $pb.ClientContext? ctx, IncrementRuntimeStateRequest request) =>
      _client.invoke<RuntimeStateResponse>(ctx, 'SharedSoDemo',
          'IncrementRuntimeState', request, RuntimeStateResponse());

  /// ReadRuntimeState observes the same state through the Kotlin/JNI path.
  $async.Future<RuntimeStateResponse> readRuntimeState(
          $pb.ClientContext? ctx, ReadRuntimeStateRequest request) =>
      _client.invoke<RuntimeStateResponse>(ctx, 'SharedSoDemo',
          'ReadRuntimeState', request, RuntimeStateResponse());

  /// WatchRuntimeState streams runtime state snapshots through the Kotlin/JNI path.
  $async.Future<RuntimeStateResponse> watchRuntimeState(
          $pb.ClientContext? ctx, ReadRuntimeStateRequest request) =>
      _client.invoke<RuntimeStateResponse>(ctx, 'SharedSoDemo',
          'WatchRuntimeState', request, RuntimeStateResponse());

  /// CollectRuntimeState verifies client streaming through both Flutter FFI and Kotlin/JNI.
  $async.Future<RuntimeStateResponse> collectRuntimeState(
          $pb.ClientContext? ctx, IncrementRuntimeStateRequest request) =>
      _client.invoke<RuntimeStateResponse>(ctx, 'SharedSoDemo',
          'CollectRuntimeState', request, RuntimeStateResponse());

  /// StreamRuntimeState verifies server streaming through both Flutter FFI and Kotlin/JNI.
  $async.Future<RuntimeStateResponse> streamRuntimeState(
          $pb.ClientContext? ctx, ReadRuntimeStateRequest request) =>
      _client.invoke<RuntimeStateResponse>(ctx, 'SharedSoDemo',
          'StreamRuntimeState', request, RuntimeStateResponse());

  /// ChatRuntimeState verifies bidi streaming through both Flutter FFI and Kotlin/JNI.
  $async.Future<RuntimeStateResponse> chatRuntimeState(
          $pb.ClientContext? ctx, IncrementRuntimeStateRequest request) =>
      _client.invoke<RuntimeStateResponse>(ctx, 'SharedSoDemo',
          'ChatRuntimeState', request, RuntimeStateResponse());
}

/// AndroidDevice exposes Android-owned capabilities through a Kotlin message server.
/// @rpccgo: msg-connect
class AndroidDeviceApi {
  final $pb.RpcClient _client;

  AndroidDeviceApi(this._client);

  /// SetTorch enables or disables the Android camera torch.
  $async.Future<SetTorchResponse> setTorch(
          $pb.ClientContext? ctx, SetTorchRequest request) =>
      _client.invoke<SetTorchResponse>(
          ctx, 'AndroidDevice', 'SetTorch', request, SetTorchResponse());

  /// WatchTorch streams Android-owned torch state observations.
  $async.Future<SetTorchResponse> watchTorch(
          $pb.ClientContext? ctx, SetTorchRequest request) =>
      _client.invoke<SetTorchResponse>(
          ctx, 'AndroidDevice', 'WatchTorch', request, SetTorchResponse());

  /// CollectTorch applies a client stream of torch requests and returns the last state.
  $async.Future<SetTorchResponse> collectTorch(
          $pb.ClientContext? ctx, SetTorchRequest request) =>
      _client.invoke<SetTorchResponse>(
          ctx, 'AndroidDevice', 'CollectTorch', request, SetTorchResponse());

  /// ChatTorch applies each torch request and streams each resulting state back.
  $async.Future<SetTorchResponse> chatTorch(
          $pb.ClientContext? ctx, SetTorchRequest request) =>
      _client.invoke<SetTorchResponse>(
          ctx, 'AndroidDevice', 'ChatTorch', request, SetTorchResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
