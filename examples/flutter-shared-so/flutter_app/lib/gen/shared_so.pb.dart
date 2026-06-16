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
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
