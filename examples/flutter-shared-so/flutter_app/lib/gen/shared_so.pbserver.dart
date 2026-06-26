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
  $async.Future<$0.RuntimeStateResponse> watchRuntimeState(
      $pb.ServerContext ctx, $0.ReadRuntimeStateRequest request);
  $async.Future<$0.RuntimeStateResponse> collectRuntimeState(
      $pb.ServerContext ctx, $0.IncrementRuntimeStateRequest request);
  $async.Future<$0.RuntimeStateResponse> streamRuntimeState(
      $pb.ServerContext ctx, $0.ReadRuntimeStateRequest request);
  $async.Future<$0.RuntimeStateResponse> chatRuntimeState(
      $pb.ServerContext ctx, $0.IncrementRuntimeStateRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'ComposeGreeting':
        return $0.ComposeGreetingRequest();
      case 'IncrementRuntimeState':
        return $0.IncrementRuntimeStateRequest();
      case 'ReadRuntimeState':
        return $0.ReadRuntimeStateRequest();
      case 'WatchRuntimeState':
        return $0.ReadRuntimeStateRequest();
      case 'CollectRuntimeState':
        return $0.IncrementRuntimeStateRequest();
      case 'StreamRuntimeState':
        return $0.ReadRuntimeStateRequest();
      case 'ChatRuntimeState':
        return $0.IncrementRuntimeStateRequest();
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
      case 'WatchRuntimeState':
        return watchRuntimeState(ctx, request as $0.ReadRuntimeStateRequest);
      case 'CollectRuntimeState':
        return collectRuntimeState(
            ctx, request as $0.IncrementRuntimeStateRequest);
      case 'StreamRuntimeState':
        return streamRuntimeState(ctx, request as $0.ReadRuntimeStateRequest);
      case 'ChatRuntimeState':
        return chatRuntimeState(
            ctx, request as $0.IncrementRuntimeStateRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      SharedSoDemoServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => SharedSoDemoServiceBase$messageJson;
}

abstract class AndroidDeviceServiceBase extends $pb.GeneratedService {
  $async.Future<$0.SetTorchResponse> setTorch(
      $pb.ServerContext ctx, $0.SetTorchRequest request);
  $async.Future<$0.AndroidEchoResponse> watchAndroidEcho(
      $pb.ServerContext ctx, $0.AndroidEchoRequest request);
  $async.Future<$0.AndroidEchoResponse> collectAndroidEcho(
      $pb.ServerContext ctx, $0.AndroidEchoRequest request);
  $async.Future<$0.AndroidEchoResponse> chatAndroidEcho(
      $pb.ServerContext ctx, $0.AndroidEchoRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'SetTorch':
        return $0.SetTorchRequest();
      case 'WatchAndroidEcho':
        return $0.AndroidEchoRequest();
      case 'CollectAndroidEcho':
        return $0.AndroidEchoRequest();
      case 'ChatAndroidEcho':
        return $0.AndroidEchoRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'SetTorch':
        return setTorch(ctx, request as $0.SetTorchRequest);
      case 'WatchAndroidEcho':
        return watchAndroidEcho(ctx, request as $0.AndroidEchoRequest);
      case 'CollectAndroidEcho':
        return collectAndroidEcho(ctx, request as $0.AndroidEchoRequest);
      case 'ChatAndroidEcho':
        return chatAndroidEcho(ctx, request as $0.AndroidEchoRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      AndroidDeviceServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => AndroidDeviceServiceBase$messageJson;
}

abstract class FlutterDeviceServiceBase extends $pb.GeneratedService {
  $async.Future<$0.FlutterEchoResponse> describeFlutter(
      $pb.ServerContext ctx, $0.FlutterEchoRequest request);
  $async.Future<$0.FlutterEchoResponse> watchFlutterEcho(
      $pb.ServerContext ctx, $0.FlutterEchoRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'DescribeFlutter':
        return $0.FlutterEchoRequest();
      case 'WatchFlutterEcho':
        return $0.FlutterEchoRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'DescribeFlutter':
        return describeFlutter(ctx, request as $0.FlutterEchoRequest);
      case 'WatchFlutterEcho':
        return watchFlutterEcho(ctx, request as $0.FlutterEchoRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      FlutterDeviceServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => FlutterDeviceServiceBase$messageJson;
}
