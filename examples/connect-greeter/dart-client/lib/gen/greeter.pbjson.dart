//
//  Generated code. Do not modify.
//  source: greeter.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use sayHelloRequestDescriptor instead')
const SayHelloRequest$json = {
  '1': 'SayHelloRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'city', '3': 2, '4': 1, '5': 9, '10': 'city'},
  ],
};

/// Descriptor for `SayHelloRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sayHelloRequestDescriptor = $convert.base64Decode(
    'Cg9TYXlIZWxsb1JlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZRISCgRjaXR5GAIgASgJUgRjaX'
    'R5');

@$core.Deprecated('Use sayHelloResponseDescriptor instead')
const SayHelloResponse$json = {
  '1': 'SayHelloResponse',
  '2': [
    {'1': 'message', '3': 1, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `SayHelloResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sayHelloResponseDescriptor = $convert.base64Decode(
    'ChBTYXlIZWxsb1Jlc3BvbnNlEhgKB21lc3NhZ2UYASABKAlSB21lc3NhZ2U=');

const $core.Map<$core.String, $core.dynamic> GreeterServiceBase$json = {
  '1': 'Greeter',
  '2': [
    {
      '1': 'SayHello',
      '2': '.examples.connect.greeter.v1.SayHelloRequest',
      '3': '.examples.connect.greeter.v1.SayHelloResponse'
    },
    {
      '1': 'Collect',
      '2': '.examples.connect.greeter.v1.SayHelloRequest',
      '3': '.examples.connect.greeter.v1.SayHelloResponse',
      '5': true
    },
    {
      '1': 'Broadcast',
      '2': '.examples.connect.greeter.v1.SayHelloRequest',
      '3': '.examples.connect.greeter.v1.SayHelloResponse',
      '6': true
    },
    {
      '1': 'Chat',
      '2': '.examples.connect.greeter.v1.SayHelloRequest',
      '3': '.examples.connect.greeter.v1.SayHelloResponse',
      '5': true,
      '6': true
    },
  ],
};

@$core.Deprecated('Use greeterServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    GreeterServiceBase$messageJson = {
  '.examples.connect.greeter.v1.SayHelloRequest': SayHelloRequest$json,
  '.examples.connect.greeter.v1.SayHelloResponse': SayHelloResponse$json,
};

/// Descriptor for `Greeter`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List greeterServiceDescriptor = $convert.base64Decode(
    'CgdHcmVldGVyEmcKCFNheUhlbGxvEiwuZXhhbXBsZXMuY29ubmVjdC5ncmVldGVyLnYxLlNheU'
    'hlbGxvUmVxdWVzdBotLmV4YW1wbGVzLmNvbm5lY3QuZ3JlZXRlci52MS5TYXlIZWxsb1Jlc3Bv'
    'bnNlEmgKB0NvbGxlY3QSLC5leGFtcGxlcy5jb25uZWN0LmdyZWV0ZXIudjEuU2F5SGVsbG9SZX'
    'F1ZXN0Gi0uZXhhbXBsZXMuY29ubmVjdC5ncmVldGVyLnYxLlNheUhlbGxvUmVzcG9uc2UoARJq'
    'CglCcm9hZGNhc3QSLC5leGFtcGxlcy5jb25uZWN0LmdyZWV0ZXIudjEuU2F5SGVsbG9SZXF1ZX'
    'N0Gi0uZXhhbXBsZXMuY29ubmVjdC5ncmVldGVyLnYxLlNheUhlbGxvUmVzcG9uc2UwARJnCgRD'
    'aGF0EiwuZXhhbXBsZXMuY29ubmVjdC5ncmVldGVyLnYxLlNheUhlbGxvUmVxdWVzdBotLmV4YW'
    '1wbGVzLmNvbm5lY3QuZ3JlZXRlci52MS5TYXlIZWxsb1Jlc3BvbnNlKAEwAQ==');
