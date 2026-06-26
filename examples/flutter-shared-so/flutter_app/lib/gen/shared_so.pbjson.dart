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

@$core.Deprecated('Use setTorchRequestDescriptor instead')
const SetTorchRequest$json = {
  '1': 'SetTorchRequest',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'caller', '3': 2, '4': 1, '5': 9, '10': 'caller'},
  ],
};

/// Descriptor for `SetTorchRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setTorchRequestDescriptor = $convert.base64Decode(
    'Cg9TZXRUb3JjaFJlcXVlc3QSGAoHZW5hYmxlZBgBIAEoCFIHZW5hYmxlZBIWCgZjYWxsZXIYAi'
    'ABKAlSBmNhbGxlcg==');

@$core.Deprecated('Use setTorchResponseDescriptor instead')
const SetTorchResponse$json = {
  '1': 'SetTorchResponse',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'camera_id', '3': 2, '4': 1, '5': 9, '10': 'cameraId'},
    {'1': 'caller', '3': 3, '4': 1, '5': 9, '10': 'caller'},
    {'1': 'status', '3': 4, '4': 1, '5': 9, '10': 'status'},
  ],
};

/// Descriptor for `SetTorchResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setTorchResponseDescriptor = $convert.base64Decode(
    'ChBTZXRUb3JjaFJlc3BvbnNlEhgKB2VuYWJsZWQYASABKAhSB2VuYWJsZWQSGwoJY2FtZXJhX2'
    'lkGAIgASgJUghjYW1lcmFJZBIWCgZjYWxsZXIYAyABKAlSBmNhbGxlchIWCgZzdGF0dXMYBCAB'
    'KAlSBnN0YXR1cw==');

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
    {
      '1': 'WatchRuntimeState',
      '2': '.examples.flutter.sharedso.v1.ReadRuntimeStateRequest',
      '3': '.examples.flutter.sharedso.v1.RuntimeStateResponse',
      '6': true
    },
    {
      '1': 'CollectRuntimeState',
      '2': '.examples.flutter.sharedso.v1.IncrementRuntimeStateRequest',
      '3': '.examples.flutter.sharedso.v1.RuntimeStateResponse',
      '5': true
    },
    {
      '1': 'StreamRuntimeState',
      '2': '.examples.flutter.sharedso.v1.ReadRuntimeStateRequest',
      '3': '.examples.flutter.sharedso.v1.RuntimeStateResponse',
      '6': true
    },
    {
      '1': 'ChatRuntimeState',
      '2': '.examples.flutter.sharedso.v1.IncrementRuntimeStateRequest',
      '3': '.examples.flutter.sharedso.v1.RuntimeStateResponse',
      '5': true,
      '6': true
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
    'VudGltZVN0YXRlUmVzcG9uc2USgAEKEVdhdGNoUnVudGltZVN0YXRlEjUuZXhhbXBsZXMuZmx1'
    'dHRlci5zaGFyZWRzby52MS5SZWFkUnVudGltZVN0YXRlUmVxdWVzdBoyLmV4YW1wbGVzLmZsdX'
    'R0ZXIuc2hhcmVkc28udjEuUnVudGltZVN0YXRlUmVzcG9uc2UwARKHAQoTQ29sbGVjdFJ1bnRp'
    'bWVTdGF0ZRI6LmV4YW1wbGVzLmZsdXR0ZXIuc2hhcmVkc28udjEuSW5jcmVtZW50UnVudGltZV'
    'N0YXRlUmVxdWVzdBoyLmV4YW1wbGVzLmZsdXR0ZXIuc2hhcmVkc28udjEuUnVudGltZVN0YXRl'
    'UmVzcG9uc2UoARKBAQoSU3RyZWFtUnVudGltZVN0YXRlEjUuZXhhbXBsZXMuZmx1dHRlci5zaG'
    'FyZWRzby52MS5SZWFkUnVudGltZVN0YXRlUmVxdWVzdBoyLmV4YW1wbGVzLmZsdXR0ZXIuc2hh'
    'cmVkc28udjEuUnVudGltZVN0YXRlUmVzcG9uc2UwARKGAQoQQ2hhdFJ1bnRpbWVTdGF0ZRI6Lm'
    'V4YW1wbGVzLmZsdXR0ZXIuc2hhcmVkc28udjEuSW5jcmVtZW50UnVudGltZVN0YXRlUmVxdWVz'
    'dBoyLmV4YW1wbGVzLmZsdXR0ZXIuc2hhcmVkc28udjEuUnVudGltZVN0YXRlUmVzcG9uc2UoAT'
    'AB');

const $core.Map<$core.String, $core.dynamic> AndroidDeviceServiceBase$json = {
  '1': 'AndroidDevice',
  '2': [
    {
      '1': 'SetTorch',
      '2': '.examples.flutter.sharedso.v1.SetTorchRequest',
      '3': '.examples.flutter.sharedso.v1.SetTorchResponse'
    },
    {
      '1': 'WatchTorch',
      '2': '.examples.flutter.sharedso.v1.SetTorchRequest',
      '3': '.examples.flutter.sharedso.v1.SetTorchResponse',
      '6': true
    },
    {
      '1': 'CollectTorch',
      '2': '.examples.flutter.sharedso.v1.SetTorchRequest',
      '3': '.examples.flutter.sharedso.v1.SetTorchResponse',
      '5': true
    },
    {
      '1': 'ChatTorch',
      '2': '.examples.flutter.sharedso.v1.SetTorchRequest',
      '3': '.examples.flutter.sharedso.v1.SetTorchResponse',
      '5': true,
      '6': true
    },
  ],
};

@$core.Deprecated('Use androidDeviceServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    AndroidDeviceServiceBase$messageJson = {
  '.examples.flutter.sharedso.v1.SetTorchRequest': SetTorchRequest$json,
  '.examples.flutter.sharedso.v1.SetTorchResponse': SetTorchResponse$json,
};

/// Descriptor for `AndroidDevice`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List androidDeviceServiceDescriptor = $convert.base64Decode(
    'Cg1BbmRyb2lkRGV2aWNlEmkKCFNldFRvcmNoEi0uZXhhbXBsZXMuZmx1dHRlci5zaGFyZWRzby'
    '52MS5TZXRUb3JjaFJlcXVlc3QaLi5leGFtcGxlcy5mbHV0dGVyLnNoYXJlZHNvLnYxLlNldFRv'
    'cmNoUmVzcG9uc2USbQoKV2F0Y2hUb3JjaBItLmV4YW1wbGVzLmZsdXR0ZXIuc2hhcmVkc28udj'
    'EuU2V0VG9yY2hSZXF1ZXN0Gi4uZXhhbXBsZXMuZmx1dHRlci5zaGFyZWRzby52MS5TZXRUb3Jj'
    'aFJlc3BvbnNlMAESbwoMQ29sbGVjdFRvcmNoEi0uZXhhbXBsZXMuZmx1dHRlci5zaGFyZWRzby'
    '52MS5TZXRUb3JjaFJlcXVlc3QaLi5leGFtcGxlcy5mbHV0dGVyLnNoYXJlZHNvLnYxLlNldFRv'
    'cmNoUmVzcG9uc2UoARJuCglDaGF0VG9yY2gSLS5leGFtcGxlcy5mbHV0dGVyLnNoYXJlZHNvLn'
    'YxLlNldFRvcmNoUmVxdWVzdBouLmV4YW1wbGVzLmZsdXR0ZXIuc2hhcmVkc28udjEuU2V0VG9y'
    'Y2hSZXNwb25zZSgBMAE=');
