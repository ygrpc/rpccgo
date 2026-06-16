// This is a generated file - do not edit.
//
// Generated from shared_so.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use composeGreetingRequestDescriptor instead')
const ComposeGreetingRequest$json = {
  '1': 'ComposeGreetingRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'caller', '3': 2, '4': 1, '5': 9, '10': 'caller'},
  ],
};

/// Descriptor for `ComposeGreetingRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List composeGreetingRequestDescriptor =
    $convert.base64Decode(
        'ChZDb21wb3NlR3JlZXRpbmdSZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWUSFgoGY2FsbGVyGA'
        'IgASgJUgZjYWxsZXI=');

@$core.Deprecated('Use composeGreetingResponseDescriptor instead')
const ComposeGreetingResponse$json = {
  '1': 'ComposeGreetingResponse',
  '2': [
    {'1': 'message', '3': 1, '4': 1, '5': 9, '10': 'message'},
    {'1': 'served_by', '3': 2, '4': 1, '5': 9, '10': 'servedBy'},
    {'1': 'library', '3': 3, '4': 1, '5': 9, '10': 'library'},
  ],
};

/// Descriptor for `ComposeGreetingResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List composeGreetingResponseDescriptor = $convert.base64Decode(
    'ChdDb21wb3NlR3JlZXRpbmdSZXNwb25zZRIYCgdtZXNzYWdlGAEgASgJUgdtZXNzYWdlEhsKCX'
    'NlcnZlZF9ieRgCIAEoCVIIc2VydmVkQnkSGAoHbGlicmFyeRgDIAEoCVIHbGlicmFyeQ==');

@$core.Deprecated('Use incrementRuntimeStateRequestDescriptor instead')
const IncrementRuntimeStateRequest$json = {
  '1': 'IncrementRuntimeStateRequest',
  '2': [
    {'1': 'delta', '3': 1, '4': 1, '5': 5, '10': 'delta'},
    {'1': 'caller', '3': 2, '4': 1, '5': 9, '10': 'caller'},
  ],
};

/// Descriptor for `IncrementRuntimeStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List incrementRuntimeStateRequestDescriptor =
    $convert.base64Decode(
        'ChxJbmNyZW1lbnRSdW50aW1lU3RhdGVSZXF1ZXN0EhQKBWRlbHRhGAEgASgFUgVkZWx0YRIWCg'
        'ZjYWxsZXIYAiABKAlSBmNhbGxlcg==');

@$core.Deprecated('Use readRuntimeStateRequestDescriptor instead')
const ReadRuntimeStateRequest$json = {
  '1': 'ReadRuntimeStateRequest',
  '2': [
    {'1': 'caller', '3': 1, '4': 1, '5': 9, '10': 'caller'},
  ],
};

/// Descriptor for `ReadRuntimeStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List readRuntimeStateRequestDescriptor =
    $convert.base64Decode(
        'ChdSZWFkUnVudGltZVN0YXRlUmVxdWVzdBIWCgZjYWxsZXIYASABKAlSBmNhbGxlcg==');

@$core.Deprecated('Use runtimeStateResponseDescriptor instead')
const RuntimeStateResponse$json = {
  '1': 'RuntimeStateResponse',
  '2': [
    {'1': 'value', '3': 1, '4': 1, '5': 3, '10': 'value'},
    {'1': 'revision', '3': 2, '4': 1, '5': 3, '10': 'revision'},
    {'1': 'instance_address', '3': 3, '4': 1, '5': 9, '10': 'instanceAddress'},
    {'1': 'caller', '3': 4, '4': 1, '5': 9, '10': 'caller'},
    {'1': 'pid', '3': 5, '4': 1, '5': 5, '10': 'pid'},
  ],
};

/// Descriptor for `RuntimeStateResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runtimeStateResponseDescriptor = $convert.base64Decode(
    'ChRSdW50aW1lU3RhdGVSZXNwb25zZRIUCgV2YWx1ZRgBIAEoA1IFdmFsdWUSGgoIcmV2aXNpb2'
    '4YAiABKANSCHJldmlzaW9uEikKEGluc3RhbmNlX2FkZHJlc3MYAyABKAlSD2luc3RhbmNlQWRk'
    'cmVzcxIWCgZjYWxsZXIYBCABKAlSBmNhbGxlchIQCgNwaWQYBSABKAVSA3BpZA==');

const $core.Map<$core.String, $core.dynamic> SharedSoDemoServiceBase$json = {
  '1': 'SharedSoDemo',
  '2': [
    {
      '1': 'ComposeGreeting',
      '2': '.examples.flutter.sharedso.v1.ComposeGreetingRequest',
      '3': '.examples.flutter.sharedso.v1.ComposeGreetingResponse'
    },
    {
      '1': 'IncrementRuntimeState',
      '2': '.examples.flutter.sharedso.v1.IncrementRuntimeStateRequest',
      '3': '.examples.flutter.sharedso.v1.RuntimeStateResponse'
    },
    {
      '1': 'ReadRuntimeState',
      '2': '.examples.flutter.sharedso.v1.ReadRuntimeStateRequest',
      '3': '.examples.flutter.sharedso.v1.RuntimeStateResponse'
    },
  ],
};

@$core.Deprecated('Use sharedSoDemoServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    SharedSoDemoServiceBase$messageJson = {
  '.examples.flutter.sharedso.v1.ComposeGreetingRequest':
      ComposeGreetingRequest$json,
  '.examples.flutter.sharedso.v1.ComposeGreetingResponse':
      ComposeGreetingResponse$json,
  '.examples.flutter.sharedso.v1.IncrementRuntimeStateRequest':
      IncrementRuntimeStateRequest$json,
  '.examples.flutter.sharedso.v1.RuntimeStateResponse':
      RuntimeStateResponse$json,
  '.examples.flutter.sharedso.v1.ReadRuntimeStateRequest':
      ReadRuntimeStateRequest$json,
};

/// Descriptor for `SharedSoDemo`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List sharedSoDemoServiceDescriptor = $convert.base64Decode(
    'CgxTaGFyZWRTb0RlbW8SfgoPQ29tcG9zZUdyZWV0aW5nEjQuZXhhbXBsZXMuZmx1dHRlci5zaG'
    'FyZWRzby52MS5Db21wb3NlR3JlZXRpbmdSZXF1ZXN0GjUuZXhhbXBsZXMuZmx1dHRlci5zaGFy'
    'ZWRzby52MS5Db21wb3NlR3JlZXRpbmdSZXNwb25zZRKHAQoVSW5jcmVtZW50UnVudGltZVN0YX'
    'RlEjouZXhhbXBsZXMuZmx1dHRlci5zaGFyZWRzby52MS5JbmNyZW1lbnRSdW50aW1lU3RhdGVS'
    'ZXF1ZXN0GjIuZXhhbXBsZXMuZmx1dHRlci5zaGFyZWRzby52MS5SdW50aW1lU3RhdGVSZXNwb2'
    '5zZRJ9ChBSZWFkUnVudGltZVN0YXRlEjUuZXhhbXBsZXMuZmx1dHRlci5zaGFyZWRzby52MS5S'
    'ZWFkUnVudGltZVN0YXRlUmVxdWVzdBoyLmV4YW1wbGVzLmZsdXR0ZXIuc2hhcmVkc28udjEuUn'
    'VudGltZVN0YXRlUmVzcG9uc2U=');
