import 'dart:ffi' as ffi;
import 'dart:io';

import '../lib/gen/greeter.greeter.rpccgo.dart';
import '../lib/gen/greeter.pb.dart' as pb;

void main(List<String> args) {
  final libraryPath =
      _argValue(args, '--library') ??
      Platform.environment['RPCCGO_CONNECT_GREETER_LIB'] ??
      _defaultLibraryPath();

  final client = GreeterRpccgoClient(ffi.DynamicLibrary.open(libraryPath));

  final unary = client.SayHello(
    pb.SayHelloRequest(name: 'dart-ffi', city: 'connect-greeter'),
  );
  print('dart unary: ${unary.message}');

  final collect = client.Collect();
  collect.send(pb.SayHelloRequest(name: 'ada', city: 'dart'));
  collect.send(pb.SayHelloRequest(name: 'grace', city: 'dart'));
  final collectResponse = collect.finish();
  print('dart collect: ${collectResponse.message}');

  final broadcast = client.Broadcast(
    pb.SayHelloRequest(name: 'stream', city: 'dart'),
  );
  print('dart broadcast: ${broadcast.read().message}');
  print('dart broadcast: ${broadcast.read().message}');
  broadcast.finish();

  final chat = client.Chat();
  for (final name in ['ada', 'grace']) {
    chat.send(pb.SayHelloRequest(name: name, city: 'dart'));
    print('dart chat: ${chat.read().message}');
  }
  chat.closeSend();
  chat.finish();
}

String? _argValue(List<String> args, String name) {
  final prefix = '$name=';
  for (var index = 0; index < args.length; index++) {
    final arg = args[index];
    if (arg.startsWith(prefix)) {
      return arg.substring(prefix.length);
    }
    if (arg == name && index + 1 < args.length) {
      return args[index + 1];
    }
  }
  return null;
}

String _defaultLibraryPath() {
  final candidates = [
    '../examples/connect-greeter/build/librpccgo_connect_greeter.so',
    'examples/connect-greeter/build/librpccgo_connect_greeter.so',
  ];
  for (final candidate in candidates) {
    if (File(candidate).existsSync()) {
      return candidate;
    }
  }
  return candidates.first;
}
