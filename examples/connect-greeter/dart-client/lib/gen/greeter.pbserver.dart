//
//  Generated code. Do not modify.
//  source: greeter.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'greeter.pb.dart' as $0;
import 'greeter.pbjson.dart';

export 'greeter.pb.dart';

abstract class GreeterServiceBase extends $pb.GeneratedService {
  $async.Future<$0.SayHelloResponse> sayHello(
      $pb.ServerContext ctx, $0.SayHelloRequest request);
  $async.Future<$0.SayHelloResponse> collect(
      $pb.ServerContext ctx, $0.SayHelloRequest request);
  $async.Future<$0.SayHelloResponse> broadcast(
      $pb.ServerContext ctx, $0.SayHelloRequest request);
  $async.Future<$0.SayHelloResponse> chat(
      $pb.ServerContext ctx, $0.SayHelloRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'SayHello':
        return $0.SayHelloRequest();
      case 'Collect':
        return $0.SayHelloRequest();
      case 'Broadcast':
        return $0.SayHelloRequest();
      case 'Chat':
        return $0.SayHelloRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'SayHello':
        return this.sayHello(ctx, request as $0.SayHelloRequest);
      case 'Collect':
        return this.collect(ctx, request as $0.SayHelloRequest);
      case 'Broadcast':
        return this.broadcast(ctx, request as $0.SayHelloRequest);
      case 'Chat':
        return this.chat(ctx, request as $0.SayHelloRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => GreeterServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => GreeterServiceBase$messageJson;
}
