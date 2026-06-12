//
//  Generated code. Do not modify.
//  source: greeter.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

/// SayHelloRequest carries the name and city used to build a greeting.
class SayHelloRequest extends $pb.GeneratedMessage {
  factory SayHelloRequest({
    $core.String? name,
    $core.String? city,
  }) {
    final $result = create();
    if (name != null) {
      $result.name = name;
    }
    if (city != null) {
      $result.city = city;
    }
    return $result;
  }
  SayHelloRequest._() : super();
  factory SayHelloRequest.fromBuffer($core.List<$core.int> i,
          [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(i, r);
  factory SayHelloRequest.fromJson($core.String i,
          [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SayHelloRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.connect.greeter.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'city')
    ..hasRequiredFields = false;

  @$core.Deprecated('Using this can add significant overhead to your binary. '
      'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
      'Will be removed in next major version')
  SayHelloRequest clone() => SayHelloRequest()..mergeFromMessage(this);
  @$core.Deprecated('Using this can add significant overhead to your binary. '
      'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
      'Will be removed in next major version')
  SayHelloRequest copyWith(void Function(SayHelloRequest) updates) =>
      super.copyWith((message) => updates(message as SayHelloRequest))
          as SayHelloRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SayHelloRequest create() => SayHelloRequest._();
  SayHelloRequest createEmptyInstance() => create();
  static $pb.PbList<SayHelloRequest> createRepeated() =>
      $pb.PbList<SayHelloRequest>();
  @$core.pragma('dart2js:noInline')
  static SayHelloRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SayHelloRequest>(create);
  static SayHelloRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String v) {
    $_setString(0, v);
  }

  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get city => $_getSZ(1);
  @$pb.TagNumber(2)
  set city($core.String v) {
    $_setString(1, v);
  }

  @$pb.TagNumber(2)
  $core.bool hasCity() => $_has(1);
  @$pb.TagNumber(2)
  void clearCity() => clearField(2);
}

/// SayHelloResponse carries the greeting text returned by the server.
class SayHelloResponse extends $pb.GeneratedMessage {
  factory SayHelloResponse({
    $core.String? message,
  }) {
    final $result = create();
    if (message != null) {
      $result.message = message;
    }
    return $result;
  }
  SayHelloResponse._() : super();
  factory SayHelloResponse.fromBuffer($core.List<$core.int> i,
          [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(i, r);
  factory SayHelloResponse.fromJson($core.String i,
          [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SayHelloResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'examples.connect.greeter.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('Using this can add significant overhead to your binary. '
      'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
      'Will be removed in next major version')
  SayHelloResponse clone() => SayHelloResponse()..mergeFromMessage(this);
  @$core.Deprecated('Using this can add significant overhead to your binary. '
      'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
      'Will be removed in next major version')
  SayHelloResponse copyWith(void Function(SayHelloResponse) updates) =>
      super.copyWith((message) => updates(message as SayHelloResponse))
          as SayHelloResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SayHelloResponse create() => SayHelloResponse._();
  SayHelloResponse createEmptyInstance() => create();
  static $pb.PbList<SayHelloResponse> createRepeated() =>
      $pb.PbList<SayHelloResponse>();
  @$core.pragma('dart2js:noInline')
  static SayHelloResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SayHelloResponse>(create);
  static SayHelloResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get message => $_getSZ(0);
  @$pb.TagNumber(1)
  set message($core.String v) {
    $_setString(0, v);
  }

  @$pb.TagNumber(1)
  $core.bool hasMessage() => $_has(0);
  @$pb.TagNumber(1)
  void clearMessage() => clearField(1);
}

class GreeterApi {
  $pb.RpcClient _client;
  GreeterApi(this._client);

  $async.Future<SayHelloResponse> sayHello(
          $pb.ClientContext? ctx, SayHelloRequest request) =>
      _client.invoke<SayHelloResponse>(
          ctx, 'Greeter', 'SayHello', request, SayHelloResponse());
  $async.Future<SayHelloResponse> collect(
          $pb.ClientContext? ctx, SayHelloRequest request) =>
      _client.invoke<SayHelloResponse>(
          ctx, 'Greeter', 'Collect', request, SayHelloResponse());
  $async.Future<SayHelloResponse> broadcast(
          $pb.ClientContext? ctx, SayHelloRequest request) =>
      _client.invoke<SayHelloResponse>(
          ctx, 'Greeter', 'Broadcast', request, SayHelloResponse());
  $async.Future<SayHelloResponse> chat(
          $pb.ClientContext? ctx, SayHelloRequest request) =>
      _client.invoke<SayHelloResponse>(
          ctx, 'Greeter', 'Chat', request, SayHelloResponse());
}

const _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
