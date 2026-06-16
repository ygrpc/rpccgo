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

import 'package:protobuf/protobuf.dart' as $pb;

import 'shared_so.pb.dart' as $0;
import 'shared_so.pbjson.dart';

export 'shared_so.pb.dart';

abstract class SharedSoDemoServiceBase extends $pb.GeneratedService {
  $async.Future<$0.ComposeGreetingResponse> composeGreeting(
      $pb.ServerContext ctx, $0.ComposeGreetingRequest request);
  $async.Future<$0.RuntimeStateResponse> incrementRuntimeState(
      $pb.ServerContext ctx, $0.IncrementRuntimeStateRequest request);
  $async.Future<$0.RuntimeStateResponse> readRuntimeState(
      $pb.ServerContext ctx, $0.ReadRuntimeStateRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'ComposeGreeting':
        return $0.ComposeGreetingRequest();
      case 'IncrementRuntimeState':
        return $0.IncrementRuntimeStateRequest();
      case 'ReadRuntimeState':
        return $0.ReadRuntimeStateRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'ComposeGreeting':
        return composeGreeting(ctx, request as $0.ComposeGreetingRequest);
      case 'IncrementRuntimeState':
        return incrementRuntimeState(
            ctx, request as $0.IncrementRuntimeStateRequest);
      case 'ReadRuntimeState':
        return readRuntimeState(ctx, request as $0.ReadRuntimeStateRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      SharedSoDemoServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => SharedSoDemoServiceBase$messageJson;
}
